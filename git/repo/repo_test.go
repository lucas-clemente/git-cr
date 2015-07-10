package repo_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/lucas-clemente/git-cr/git/repo"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocalRepo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JSON Repo Suite")
}

type fixtureBackend map[string][]byte

func (f fixtureBackend) ReadBlob(name string) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBuffer(f[name])), nil
}

func (f fixtureBackend) WriteBlob(name string, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	f[name] = data
	return nil
}

var _ = Describe("JSON Repo", func() {
	var (
		backend  fixtureBackend
		jsonRepo repo.Repo
	)

	BeforeEach(func() {
		backend = fixtureBackend{}
		jsonRepo = repo.NewJSONRepo(backend)
	})

	It("reads packfiles", func() {
		backend["42.pack"] = []byte("foo")
		r, err := jsonRepo.ReadPackfile(42)
		Ω(err).ShouldNot(HaveOccurred())
		data, err := ioutil.ReadAll(r)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(data).Should(Equal([]byte("foo")))
	})

	It("reads revisions", func() {
		backend["revisions.json"] = []byte(`[{"refs/heads/master":"foobar"}]`)
		refs, err := jsonRepo.GetRevisions()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(refs).Should(Equal([]repo.Revision{{"refs/heads/master": "foobar"}}))
	})

	It("saves new revisions", func() {
		backend["revisions.json"] = []byte(`[{"refs/heads/master":"foobar"}]`)
		err := jsonRepo.SaveNewRevision(repo.Revision{"refs/heads/master": "foobaz"}, bytes.NewBufferString("bar"))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(backend["revisions.json"]).Should(Equal([]byte(`[{"refs/heads/master":"foobar"},{"refs/heads/master":"foobaz"}]`)))
		Ω(backend["1.pack"]).Should(Equal([]byte("bar")))
	})
})
