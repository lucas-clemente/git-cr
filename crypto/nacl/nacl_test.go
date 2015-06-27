package nacl_test

import (
	"bytes"
	"io/ioutil"
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

	Context("encrypting and decrypting refs", func() {
		It("encrypts refs", func() {
			err = repo.WriteRefs(bytes.NewBufferString(`{"foo":"bar"}`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(backend.CurrentRefs).ShouldNot(HaveLen(0))
			Ω(backend.CurrentRefs).ShouldNot(ContainSubstring(`{"foo":"bar"}`))
		})

		It("reads encrypted refs", func() {
			backend.CurrentRefs = []byte{0x2e, 0xa, 0x12, 0xb7, 0xd5, 0xff, 0xdd, 0xe0,
				0xcf, 0x3e, 0x17, 0x46, 0x5e, 0x39, 0x4f, 0x17, 0xd1, 0xa, 0x67, 0x59,
				0x2a, 0xa3, 0xdd, 0xc6, 0x6a, 0x91, 0x3, 0x84, 0xae, 0x83, 0xb0, 0x37,
				0xc7, 0x8b, 0xfd, 0x7a, 0x8b, 0x93, 0xfb, 0x3f, 0x74, 0x1c, 0xb, 0xe5,
				0x3a, 0x10, 0x73, 0xb3, 0xe8, 0x25, 0x80, 0xaa, 0x87}
			r, err := repo.ReadRefs()
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadAll(r)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(Equal([]byte(`{"foo":"bar"}`)))
		})

		It("errors on invalid encrypted data", func() {
			backend.CurrentRefs = []byte{0x2e, 0xa, 0x12, 0xb7, 0xd5, 0xff, 0xdd, 0xe0,
				0xcf, 0x3e, 0x17, 0x46, 0x5e, 0x39, 0x4f, 0x17, 0xd1, 0xa, 0x67, 0x59,
				0x2a, 0xa3, 0xdd, 0xc6, 0x6a, 0x91, 0x3, 0x84, 0xae, 0x83, 0xb0, 0x37,
				0xc7, 0x8b, 0xfd, 0x7a, 0x8b, 0x93, 0xfb, 0x3f, 0x74, 0x1c, 0xb, 0xe5,
				0x3a, 0x10, 0x73, 0xb3, 0xe8, 0x25, 0x80, 0xaa, 0x88}
			_, err := repo.ReadRefs()
			Ω(err).Should(HaveOccurred())
		})

		It("encrypts and decrypts refs", func() {
			err = repo.WriteRefs(bytes.NewBufferString(`{"foo":"bar"}`))
			Ω(err).ShouldNot(HaveOccurred())
			r, err := repo.ReadRefs()
			Ω(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadAll(r)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(Equal([]byte(`{"foo":"bar"}`)))
		})

	})
})