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
	"github.com/lucas-clemente/git-cr/git/handler"
	"github.com/lucas-clemente/git-cr/git/repo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

func fillRepo(b *FixtureRepo) {
	b.SaveNewRevisionB64(
		repo.Revision{
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
		tempDir     string
		fixtureRepo *FixtureRepo
		server      *handler.GitRequestHandler
		listener    net.Listener
		port        string
		mutex       sync.Mutex
	)

	BeforeEach(func() {
		var err error

		mutex = sync.Mutex{}

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		fixtureRepo = NewFixtureRepo()

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

				server = handler.NewGitRequestHandler(encoder, decoder, fixtureRepo)
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
			fillRepo(fixtureRepo)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/fixtureRepo", ".")
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("bar\n")))

			mutex.Lock()
			mutex.Unlock()
		})

		It("clones multiple references", func() {
			fillRepo(fixtureRepo)
			fixtureRepo.SaveNewRevisionB64(
				repo.Revision{
					"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
					"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
					"refs/heads/foobar": "226b4f2fd9f8ca09f9abe37612c06fe4527694f5",
				},
				"UEFDSwAAAAIAAAADnAp4nJ3LwQrCMAwA0Hu/IndB0qZpEUQEr/uJtKY6WC1s2f+LsC/w+A7PVlVoyNJCpiKUmrLPSVlFCsVLSl44FXqGLOhkt/dYYdqrbPBYtOvHFK7Lz/d6+DyPG/hInMlTjnDCgOjq6H020/+269vLfQEVLTSCMHicAwAAAAABoAJ4nDM0MDAzMVEoSS0uYXg299HsTRevOXt3a64rj7px6ElP8EQA1EMPGJoJJjoehuEy+kV9XYBCyAkBMpTu",
			)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/fixtureRepo", ".")
		})
	})

	Context("pulling", func() {
		BeforeEach(func() {
			fillRepo(fixtureRepo)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/fixtureRepo", ".")
		})

		It("pulls updates", func() {
			fixtureRepo.SaveNewRevisionB64(
				repo.Revision{
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
			fillRepo(fixtureRepo)
			runCommandInDir(tempDir, "git", "clone", "git://localhost:"+port+"/fixtureRepo", ".")
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
			Ω(fixtureRepo.Revisions).Should(HaveLen(2))
			Ω(fixtureRepo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(fixtureRepo.Revisions[1]).Should(Equal(repo.Revision{
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

			Ω(fixtureRepo.Revisions).Should(HaveLen(2))
			Ω(fixtureRepo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(fixtureRepo.Revisions[1]).Should(Equal(repo.Revision{
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

			Ω(fixtureRepo.Revisions).Should(HaveLen(3))
			Ω(fixtureRepo.Packfiles[2]).ShouldNot(HaveLen(0))
			Ω(fixtureRepo.Revisions[2]).Should(Equal(repo.Revision{
				"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
				"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
			}))

			// Clone again
			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", "git://localhost:"+port+"/fixtureRepo", workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("git", "branch")
			cmd.Dir = workingDir2
			Ω(cmd.CombinedOutput()).Should(Equal([]byte("* master\n")))
		})

		It("pushes empty updates", func() {
			runCommandInDir(tempDir, "git", "push", "origin")
			Ω(fixtureRepo.Revisions).Should(HaveLen(1))
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

			Ω(fixtureRepo.Revisions).Should(HaveLen(2))
			Ω(fixtureRepo.Packfiles[1]).ShouldNot(HaveLen(0))
			Ω(fixtureRepo.Revisions[1]).Should(HaveKey("refs/heads/master"))
			Ω(fixtureRepo.Revisions[1]).Should(HaveKey("refs/heads/foobar"))

			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", "git://localhost:"+port+"/fixtureRepo", workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("pushing into empty fixtureRepos", func() {
		It("works", func() {
			runCommandInDir(tempDir, "git", "init")
			configGit(tempDir)

			err := ioutil.WriteFile(tempDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())

			runCommandInDir(tempDir, "git", "add", "foo")
			runCommandInDir(tempDir, "git", "commit", "-m", "test")
			runCommandInDir(tempDir, "git", "remote", "add", "origin", "git://localhost:"+port+"/fixtureRepo")
			runCommandInDir(tempDir, "git", "push", "origin", "master")

			mutex.Lock()
			mutex.Unlock()

			// Clone into second dir
			tempDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(tempDir2)

			err = exec.Command("git", "clone", "git://localhost:"+port+"/fixtureRepo", tempDir2).Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir2 + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("foobar")))
		})
	})
})
