package git_test

import (
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/bargez/pktline"
	"github.com/lucas-clemente/git-cr/git"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

var _ = Describe("integration with git", func() {
	var (
		tempDir  string
		backend  *fixtureBackend
		server   *git.GitRequestHandler
		listener net.Listener
		port     string
	)

	BeforeEach(func() {
		var err error

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		backend = newFixtureBackend()
		backend.currentRefs = git.Refs{
			"HEAD":              "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
			"refs/heads/master": "f84b0d7375bcb16dd2742344e6af173aeebfcfd6",
		}
		backend.addPackfile("", "f84b0d7375bcb16dd2742344e6af173aeebfcfd6", "UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=")

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

				encoder := pktline.NewEncoder(conn)
				decoder := &pktlineDecoderWrapper{Decoder: pktline.NewDecoder(conn), Reader: conn}

				server = git.NewGitRequestHandler(encoder, decoder, backend)
				err = server.ServeRequest()
				Ω(err).ShouldNot(HaveOccurred())
				conn.Close()
			}
		}()

	})

	AfterEach(func() {
		listener.Close()
		os.RemoveAll(tempDir)
	})

	Context("cloning", func() {
		It("clones using git", func() {
			err := exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir).Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("bar\n")))
		})
	})

	Context("pulling", func() {
		BeforeEach(func() {
			err := exec.Command("git", "clone", "git://localhost:"+port+"/repo", tempDir).Run()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("pulls updates", func() {
			backend.currentRefs["HEAD"] = "1a6d946069d483225913cf3b8ba8eae4c894c322"
			backend.currentRefs["refs/heads/master"] = "1a6d946069d483225913cf3b8ba8eae4c894c322"
			backend.addPackfile("f84b0d7375bcb16dd2742344e6af173aeebfcfd6", "1a6d946069d483225913cf3b8ba8eae4c894c322", "UEFDSwAAAAIAAAADlgx4nJXLSwrCMBRG4XlWkbkgSe5NbgpS3Eoef1QwtrQRXL51CU7O4MA3NkDnmqgFT0CSBhIGI0RhmeBCCb5Mk2cbWa1pw2voFjmbKiQ+l2xDrU7YER8oNSuUgNxKq0Gl97gvmx7Yh778esUn9fWJc1n6rC0TG0suOn0yzhh13P4YA38Q1feb+gIlsDr0M3icS0qsAgACZQE+rwF4nDM0MDAzMVFIy89nsJ9qkZYUaGwfv1Tygdym9MuFp+ZUAACUGAuBskz7fFz81Do1iG8hcUrj/ncK63Q=")
			cmd := exec.Command("git", "pull")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			contents, err := ioutil.ReadFile(tempDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("baz")))
		})
	})

	Context("pushing", func() {
		BeforeEach(func() {
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
			cmd = exec.Command("git", "config", "user.name", "test")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			cmd = exec.Command("git", "config", "user.email", "test@example.com")
			cmd.Dir = tempDir
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
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
			Ω(backend.pushedRevs).Should(Equal([]string{"1a6d946069d483225913cf3b8ba8eae4c894c322"}))
			Ω(backend.updatedRefs).Should(HaveLen(1))
			Ω(backend.updatedRefs[0].Name).Should(Equal("refs/heads/master"))
			Ω(backend.updatedRefs[0].OldID).Should(Equal("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
			Ω(backend.updatedRefs[0].NewID).Should(Equal("1a6d946069d483225913cf3b8ba8eae4c894c322"))
		})

		It("pushes deletes", func() {
			// Push
			cmd := exec.Command("git", "push", "origin", ":master")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Verify
			Ω(backend.updatedRefs).Should(HaveLen(1))
			Ω(backend.updatedRefs[0].Name).Should(Equal("refs/heads/master"))
			Ω(backend.updatedRefs[0].OldID).Should(Equal("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
			Ω(backend.updatedRefs[0].NewID).Should(Equal(""))
		})

		It("pushes new branches", func() {
			// Push
			cmd := exec.Command("git", "push", "origin", "master:foobar")
			cmd.Dir = tempDir
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())
			// Verify
			Ω(backend.updatedRefs).Should(HaveLen(1))
			Ω(backend.updatedRefs[0].Name).Should(Equal("refs/heads/foobar"))
			Ω(backend.updatedRefs[0].OldID).Should(Equal(""))
			Ω(backend.updatedRefs[0].NewID).Should(Equal("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
		})
	})
})
