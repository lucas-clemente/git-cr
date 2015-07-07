package local_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/lucas-clemente/git-cr/backends/local"
	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocalRepo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Backend Suite")
}

var _ = Describe("Local Backend", func() {
	var (
		tmpDir  string
		backend git.Backend
	)

	BeforeEach(func() {
		var err error

		tmpDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		backend, err = local.NewLocalBackend(tmpDir)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("reads", func() {
		err := ioutil.WriteFile(tmpDir+"/foo", []byte("bar"), 0644)
		Ω(err).ShouldNot(HaveOccurred())
		r, err := backend.ReadBlob("foo")
		Ω(err).ShouldNot(HaveOccurred())
		data, err := ioutil.ReadAll(r)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(data).Should(Equal([]byte("bar")))
	})

	It("writes", func() {
		err := backend.WriteBlob("foo", bytes.NewBufferString("bar"))
		Ω(err).ShouldNot(HaveOccurred())
		data, err := ioutil.ReadFile(tmpDir + "/foo")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(data).Should(Equal([]byte("bar")))
	})
})
