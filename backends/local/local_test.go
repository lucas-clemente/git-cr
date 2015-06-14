package local_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/lucas-clemente/git-cr/backends/local"
	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocalBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Backend Suite")
}

var _ = Describe("Local Backend", func() {
	var (
		tmpDir  string
		backend git.ListingBackend
	)

	BeforeEach(func() {
		var err error

		tmpDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())

		backend = local.NewLocalBackend(tmpDir)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("getting refs", func() {
		It("works", func() {
			err := ioutil.WriteFile(tmpDir+"/refs.json", []byte(`{"HEAD": "foobar","refs/heads/master":"foobar"}`), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			refs, err := backend.GetRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal(git.Refs{
				"HEAD":              "foobar",
				"refs/heads/master": "foobar",
			}))
		})

		It("errors properly on new repos", func() {
			_, err := backend.GetRefs()
			Ω(err).ShouldNot(BeNil())
			Ω(os.IsNotExist(err)).Should(BeTrue())
		})
	})
})
