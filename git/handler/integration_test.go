package handler_test

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
	"github.com/lucas-clemente/git-cr/git/handler"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

func fillRepo(b *FixtureRepo) {
	b.SaveNewRevisionB64(
		git.Revision{
			"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
			"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
		},
		"UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=",
	)
}

func runCommandInDir(dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	err := cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
}

func configGit(dir string) {
	runCommandInDir(dir, "git", "config", "user.name", "test")
	runCommandInDir(dir, "git", "config", "user.email", "test@example.com")
}

var _ = Describe("integration with git", func() {
	var (
		tempDir  string
		repo     *FixtureRepo
		server   *handler.GitRequestHandler
		listener net.Listener
		port     string
		mutex    sync.Mutex
	)

	BeforeEach(func() {
		var err error

		mutex = sync.Mutex{}

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		repo = NewFixtureRepo()

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

				server = handler.NewGitRequestHandler(encoder, decoder, repo)
				err = server.ServeRequest()
				if err != nil {
					fmt.Println("error in integration test: ", err.Error())
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
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/repo", ".")
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("bar\n")))

			mutex.Lock()
			mutex.Unlock()
		})

		It("clones multiple references", func() {
			fillRepo(repo)
			repo.SaveNewRevisionB64(
				git.Revision{
					"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
					"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
					"refs/heads/foobar": "226b4f2fd9f8ca09f9abe37612c06fe4527694f5",
				},
				"UEFDSwAAAAIAAAADnAp4nJ3LwQrCMAwA0Hu/IndB0qZpEUQEr/uJtKY6WC1s2f+LsC/w+A7PVlVoyNJCpiKUmrLPSVlFCsVLSl44FXqGLOhkt/dYYdqrbPBYtOvHFK7Lz/d6+DyPG/hInMlTjnDCgOjq6H020/+269vLfQEVLTSCMHicAwAAAAABoAJ4nDM0MDAzMVEoSS0uYXg299HsTRevOXt3a64rj7px6ElP8EQA1EMPGJoJJjoehuEy+kV9XYBCyAkBMpTu",
			)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/repo", ".")
		})
	})

	Context("pulling", func() {
		BeforeEach(func() {
			fillRepo(repo)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/repo", ".")
		})

		It("pulls updates", func() {
			repo.SaveNewRevisionB64(
				git.Revision{
					"HEAD":              "1a6d946069d483225913cf3b8ba8eae4c894c322",
					"refs/heads/master": "1a6d946069d483225913cf3b8ba8eae4c894c322",
				},
				"UEFDSwAAAAIAAAADlgx4nJXLSwrCMBRG4XlWkbkgSe5NbgpS3Eoef1QwtrQRXL51CU7O4MA3NkDnmqgFT0CSBhIGI0RhmeBCCb5Mk2cbWa1pw2voFjmbKiQ+l2xDrU7YER8oNSuUgNxKq0Gl97gvmx7Yh778esUn9fWJc1n6rC0TG0suOn0yzhh13P4YA38Q1feb+gIlsDr0M3icS0qsAgACZQE+rwF4nDM0MDAzMVFIy89nsJ9qkZYUaGwfv1Tygdym9MuFp+ZUAACUGAuBskz7fFz81Do1iG8hcUrj/ncK63Q=",
			)
			runCommandInDir(tempDir, "git", "pull")
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("baz")))
		})

		It("pulls nothing", func() {
			runCommandInDir(tempDir, "git", "pull")
		})

		It("lists remote refs", func() {
			cmd := exec.Command("git", "ls-remote")
			cmd.Dir = tempDir
			out, err := cmd.CombinedOutput()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(out).Should(ContainSubstring("refs/heads/master"))
			Ω(out).Should(ContainSubstring("HEAD"))
			Ω(out).Should(ContainSubstring("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
		})
	})

	Context("pushing changes", func() {
		BeforeEach(func() {
			fillRepo(repo)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/repo", ".")
		})

		It("pushes updates", func() {
			// Modify file
			err := ioutil.WriteFile(tempDir+"/foo", []byte("baz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			// Add
			runCommandInDir(tempDir, "git", "add", "foo")
			// Settings
			configGit(tempDir)
			// Commit
			cmd := exec.Command("git", "commit", "--message=msg")
			cmd.Dir = tempDir
			cmd.Env = []string{
				"GIT_COMMITTER_DATE=Thu Jun 11 11:01:22 2015 +0200",
				"GIT_AUTHOR_DATE=Thu Jun 11 11:01:22 2015 +0200",
			}
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Push
			runCommandInDir(tempDir, "git", "push")
			// Verify
			mutex.Lock()
			mutex.Unlock()
			Ω(repo.Revisions).Should(HaveLen(2))
			Ω(repo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(repo.Revisions[1]).Should(Equal(git.Revision{
				"refs/heads/master": "1a6d946069d483225913cf3b8ba8eae4c894c322",
				"HEAD":              "1a6d946069d483225913cf3b8ba8eae4c894c322",
			}))
		})

		It("pushes new branches", func() {
			// Push
			runCommandInDir(tempDir, "git", "push", "origin", "master:foobar")
			// Verify
			mutex.Lock()
			mutex.Unlock()

			Ω(repo.Revisions).Should(HaveLen(2))
			Ω(repo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(repo.Revisions[1]).Should(Equal(git.Revision{
				"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
				"refs/heads/foobar": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
				"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
			}))
		})

		It("pushes deletes", func() {
			// Push
			runCommandInDir(tempDir, "git", "push", "origin", "master:foobar")
			runCommandInDir(tempDir, "git", "push", "origin", ":foobar")
			// Verify
			mutex.Lock()
			mutex.Unlock()

			Ω(repo.Revisions).Should(HaveLen(3))
			Ω(repo.Packfiles[2]).ShouldNot(HaveLen(0))
			Ω(repo.Revisions[2]).Should(Equal(git.Revision{
				"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
				"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
			}))

			// Clone again
			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", "git://localhost:"+port+"/repo", workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("git", "branch")
			cmd.Dir = workingDir2
			Ω(cmd.CombinedOutput()).Should(Equal([]byte("* master\n")))
		})

		It("pushes empty updates", func() {
			runCommandInDir(tempDir, "git", "push", "origin")
			Ω(repo.Revisions).Should(HaveLen(1))
		})

		It("pushes multiple refs at once", func() {
			configGit(tempDir)

			err := ioutil.WriteFile(tempDir+"/foo", []byte("baz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(tempDir, "git", "add", "foo")
			runCommandInDir(tempDir, "git", "commit", "-m", "msg")

			runCommandInDir(tempDir, "git", "checkout", "-b", "foobar", "HEAD^")

			err = ioutil.WriteFile(tempDir+"/bar", []byte("baz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(tempDir, "git", "add", "bar")
			runCommandInDir(tempDir, "git", "commit", "-m", "msg2")

			runCommandInDir(tempDir, "git", "push", "--all")

			mutex.Lock()
			mutex.Unlock()

			Ω(repo.Revisions).Should(HaveLen(2))
			Ω(repo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(repo.Revisions[1]).Should(HaveKey("refs/heads/master"))
			Ω(repo.Revisions[1]).Should(HaveKey("refs/heads/foobar"))

			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", "git://localhost:"+port+"/repo", workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("pushing into empty repos", func() {
		It("works", func() {
			runCommandInDir(tempDir, "git", "init")
			configGit(tempDir)

			err := ioutil.WriteFile(tempDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())

			runCommandInDir(tempDir, "git", "add", "foo")
			runCommandInDir(tempDir, "git", "commit", "-m", "test")
			runCommandInDir(tempDir, "git", "remote", "add", "origin", "git://localhost:"+port+"/repo")
			runCommandInDir(tempDir, "git", "push", "origin", "master")

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
