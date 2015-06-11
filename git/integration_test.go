package git_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"

	"github.com/bargez/pktline"
	"github.com/lucas-clemente/git-cr/git"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const port = "6758"

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

type fixtureBackend struct {
	updatedRefs     []git.RefUpdate
	pushedPackfiles [][]byte
	pushedRevs      []string
}

var _ git.Backend = &fixtureBackend{}

func (*fixtureBackend) DeltaFromZero(id string) (git.Delta, error) {
	if id == "f84b0d7375bcb16dd2742344e6af173aeebfcfd6" {
		pack, err := base64.StdEncoding.DecodeString(
			"UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=",
		)
		Ω(err).ShouldNot(HaveOccurred())
		return ioutil.NopCloser(bytes.NewBuffer(pack)), nil
	}
	panic("delta from 0 not found")
}

func (*fixtureBackend) FindDelta(from, to string) (git.Delta, error) {
	panic("find delta")
}

func (*fixtureBackend) GetRefs() ([]git.Ref, error) {
	return []git.Ref{
		git.Ref{Name: "HEAD", Sha1: "f84b0d7375bcb16dd2742344e6af173aeebfcfd6"},
		git.Ref{Name: "refs/heads/master", Sha1: "f84b0d7375bcb16dd2742344e6af173aeebfcfd6"},
	}, nil
}

func (*fixtureBackend) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	return d.(io.ReadCloser), nil
}

func (b *fixtureBackend) UpdateRef(update git.RefUpdate) error {
	b.updatedRefs = append(b.updatedRefs, update)
	return nil
}

func (b *fixtureBackend) WritePackfile(from, to string, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	b.pushedPackfiles = append(b.pushedPackfiles, data)
	b.pushedRevs = append(b.pushedRevs, to)
	return nil
}

var _ = Describe("integration with git", func() {
	var (
		tempDir  string
		backend  *fixtureBackend
		server   *git.GitServer
		listener net.Listener
	)

	BeforeEach(func() {
		var err error

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		backend = &fixtureBackend{
			updatedRefs:     []git.RefUpdate{},
			pushedPackfiles: [][]byte{},
			pushedRevs:      []string{},
		}

		listener, err = net.Listen("tcp", "localhost:"+port)
		Ω(err).ShouldNot(HaveOccurred())

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

				server = git.NewGitServer(encoder, decoder, backend)
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
