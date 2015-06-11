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

type fixtureDelta io.ReadCloser

type fixtureBackend struct {
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

		backend = &fixtureBackend{}

		listener, err = net.Listen("tcp", "localhost:"+port)
		Ω(err).ShouldNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()

			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			encoder := pktline.NewEncoder(conn)
			decoder := pktline.NewDecoder(conn)

			server = git.NewGitServer(encoder, decoder, backend)
			err = server.ServeRequest()
			Ω(err).ShouldNot(HaveOccurred())
			conn.Close()
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
})
