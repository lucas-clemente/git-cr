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

	Context("updating refs", func() {
		It("creates new repos", func() {
			err := backend.UpdateRef(git.RefUpdate{Name: "refs/heads/master", NewID: "988881adc9fc3655077dc2d4d757d480b5ea0e11"})
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadFile(tmpDir + "/refs.json")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(MatchJSON(`{"refs/heads/master": "988881adc9fc3655077dc2d4d757d480b5ea0e11"}`))
		})

		It("handles deletes", func() {
			err := ioutil.WriteFile(tmpDir+"/refs.json", []byte(`{"HEAD": "foobar","refs/heads/master":"foobar"}`), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			err = backend.UpdateRef(git.RefUpdate{Name: "refs/heads/master", NewID: ""})
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadFile(tmpDir + "/refs.json")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(MatchJSON(`{"HEAD": "foobar"}`))
		})

		It("handles updates", func() {
			err := ioutil.WriteFile(tmpDir+"/refs.json", []byte(`{"HEAD": "foobar","refs/heads/master":"foobar"}`), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			err = backend.UpdateRef(git.RefUpdate{Name: "refs/heads/master", NewID: "barfoo"})
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadFile(tmpDir + "/refs.json")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(MatchJSON(`{"HEAD": "foobar","refs/heads/master":"barfoo"}`))
		})
	})

	Context("writing packfiles", func() {
		It("works", func() {
			packfileReader := bytes.NewBufferString("foobar")
			err := backend.WritePackfile("from", "to", packfileReader)
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadFile(tmpDir + "/from_to.pack")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(data)).Should(Equal("foobar"))
		})
	})

	Context("getting deltas", func() {
		It("works", func() {
			err := ioutil.WriteFile(tmpDir+"/from_to.pack", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			delta, err := backend.FindDelta("from", "to")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(delta).ShouldNot(BeNil())
		})
	})

	Context("reading packfiles", func() {
		It("works", func() {
			err := ioutil.WriteFile(tmpDir+"/from_to.pack", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			delta, err := backend.FindDelta("from", "to")
			Ω(err).ShouldNot(HaveOccurred())
			r, err := backend.ReadPackfile(delta)
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadAll(r)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(data)).Should(Equal("foobar"))
		})
	})

	Context("listing ancestors", func() {
		It("works", func() {
			err := ioutil.WriteFile(tmpDir+"/foo_baz.pack", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			err = ioutil.WriteFile(tmpDir+"/foo_bar.pack", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			err = ioutil.WriteFile(tmpDir+"/fuu_bar.pack", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			ancestors, err := backend.ListAncestors("bar")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ancestors).Should(Equal([]string{"foo", "fuu"}))
		})

		It("returns nil slice", func() {
			ancestors, err := backend.ListAncestors("bar")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ancestors).Should(HaveLen(0))
		})
	})
})
