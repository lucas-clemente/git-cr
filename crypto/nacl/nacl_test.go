package nacl_test

import (
	"bytes"
	"testing"

	"github.com/lucas-clemente/git-cr/crypto/nacl"
	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/repos/fixture"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNacl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nacl Suite")
}

var _ = Describe("Nacl", func() {
	var (
		repo    git.Repo
		backend *fixture.FixtureRepo
		key     [32]byte
		err     error
	)

	BeforeEach(func() {
		copy(key[:], "Forty-two, said Deep Thought, with infinite majesty and calm.")
		Ω(err).ShouldNot(HaveOccurred())
		backend = fixture.NewFixtureRepo()
		repo, err = nacl.NewNaclRepo(backend, key)
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("encrypts refs", func() {
		err = repo.WriteRefs(bytes.NewBufferString(`{"foo":"bar"}`))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(backend.CurrentRefs).ShouldNot(HaveLen(0))
		Ω(backend.CurrentRefs).ShouldNot(ContainSubstring(`{"foo":"bar"}`))
	})
})
