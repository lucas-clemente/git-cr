package git_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/bargez/pktline"
	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/repos/fixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

func fillRepo(b *fixture.FixtureRepo) {
	b.CurrentRefs = git.Refs{
		"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
		"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
	}
	b.AddPackfile("", "f84b0d7375bcb16dd2742344e6af173aeebfcfd6", "UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=")
}

func configGit(dir string) {
	cmd := exec.Command("git", "config", "user.name", "test")
	cmd.Dir = dir
	err := cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	err = cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
}

var _ = Describe("integration with git", func() {
	var (
		tempDir  string
		repo     *fixture.FixtureRepo
		server   *git.GitRequestHandler
		listener net.Listener
		port     string
		mutex    sync.Mutex
	)

	BeforeEach(func() {
		var err error

		mutex = sync.Mutex{}

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		repo = fixture.NewFixtureRepo()

		listener, err = net.Listen("tcp", "localhost:0")
		Ω(err).ShouldNot(HaveOccurred())
		port = strings.Split(listener.Addr().String(), ":")[1]

		go func() {
			defer GinkgoRecover()

			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()

				mutex.Lock()

				encoder := pktline.NewEncoder(conn)
				decoder := &pktlineDecoderWrapper{Decoder: pktline.NewDecoder(conn), Reader: conn}

				server = git.NewGitRequestHandler(encoder, decoder, repo)
				err = server.ServeRequest()
				if err != nil {
					fmt.Println(err.Error())
				}
				Ω(err).ShouldNot(HaveOccurred())
				conn.Close()

				mutex.Unlock()
			}
		}()

	})

	AfterEach(func() {
		listener.Close()
		mutex.Lock()
		mutex.Unlock()
		os.RemoveAll(tempDir)
	})

	Context("cloning", func() {
		It("clones using git", func() {
			fillRepo(repo)
			err := exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir).Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("bar\n")))

			mutex.Lock()
			mutex.Unlock()
		})
	})

	Context("pulling", func() {
		BeforeEach(func() {
			fillRepo(repo)
			err := exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir).Run()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("pulls updates", func() {
			repo.CurrentRefs["HEAD"] = "1a6d946069d483225913cf3b8ba8eae4c894c322"
			repo.CurrentRefs["refs/heads/master"] = "1a6d946069d483225913cf3b8ba8eae4c894c322"
			repo.AddPackfile("f84b0d7375bcb16dd2742344e6af173aeebfcfd6", "1a6d946069d483225913cf3b8ba8eae4c894c322", "UEFDSwAAAAIAAAADlgx4nJXLSwrCMBRG4XlWkbkgSe5NbgpS3Eoef1QwtrQRXL51CU7O4MA3NkDnmqgFT0CSBhIGI0RhmeBCCb5Mk2cbWa1pw2voFjmbKiQ+l2xDrU7YER8oNSuUgNxKq0Gl97gvmx7Yh778esUn9fWJc1n6rC0TG0suOn0yzhh13P4YA38Q1feb+gIlsDr0M3icS0qsAgACZQE+rwF4nDM0MDAzMVFIy89nsJ9qkZYUaGwfv1Tygdym9MuFp+ZUAACUGAuBskz7fFz81Do1iG8hcUrj/ncK63Q=")
			cmd := exec.Command("git", "pull")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("baz")))
		})
	})

	Context("pushing changes", func() {
		BeforeEach(func() {
			fillRepo(repo)
			err := exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir).Run()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("pushes updates", func() {
			// Modify file
			err := ioutil.WriteFile(tempDir+"/foo", []byte("baz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			// Add
			cmd := exec.Command("git", "add", "foo")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Settings
			configGit(tempDir)
			// Commit
			cmd = exec.Command("git", "commit", "--message=msg")
			cmd.Dir = tempDir
			cmd.Env = []string{
				"GIT_COMMITTER_DATE=Thu Jun 11 11:01:22 2015 +0200",
				"GIT_AUTHOR_DATE=Thu Jun 11 11:01:22 2015 +0200",
			}
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Push
			cmd = exec.Command("git", "push")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Verify
			mutex.Lock()
			mutex.Unlock()
			Ω(repo.PackfilesFromTo["f84b0d7375bcb16dd2742344e6af173aeebfcfd6"]["1a6d946069d483225913cf3b8ba8eae4c894c322"]).ShouldNot(HaveLen(0))
			Ω(repo.CurrentRefs).Should(HaveLen(2))
			Ω(repo.CurrentRefs["refs/heads/master"]).Should(Equal("1a6d946069d483225913cf3b8ba8eae4c894c322"))
		})

		It("pushes deletes", func() {
			// Push
			cmd := exec.Command("git", "push", "origin", ":master")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Verify
			mutex.Lock()
			mutex.Unlock()
			Ω(repo.CurrentRefs).Should(HaveLen(1))
		})

		It("pushes new branches", func() {
			// Push
			cmd := exec.Command("git", "push", "origin", "master:foobar")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Verify
			mutex.Lock()
			mutex.Unlock()
			Ω(repo.CurrentRefs).Should(HaveLen(3))
			Ω(repo.CurrentRefs["refs/heads/foobar"]).Should(Equal("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
		})
	})

	Context("pushing into empty repos", func() {
		It("works", func() {
			cmd := exec.Command("git", "init")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			err = ioutil.WriteFile(tempDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("git", "add", "foo")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			configGit(tempDir)

			cmd = exec.Command("git", "commit", "-m", "test")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("git", "remote", "add", "origin", "git://localhost:"+port+"/repo")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("git", "push", "origin", "master")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			mutex.Lock()
			mutex.Unlock()

			// Clone into second dir
			tempDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(tempDir2)

			err = exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir2).Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir2 + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("foobar")))
		})
	})
})
