package nacl_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/lucas-clemente/git-cr/crypto/nacl"
	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNaCl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NaCl Suite")
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

var _ = Describe("NaCl", func() {
	var (
		naclBackend git.Backend
		backend     fixtureBackend
		key         [32]byte
		err         error
	)

	BeforeEach(func() {
		copy(key[:], "Forty-two, said Deep Thought, with infinite majesty and calm.")
		Ω(err).ShouldNot(HaveOccurred())
		backend = fixtureBackend{}
		naclBackend = nacl.NewNaClBackend(backend, key)
	})

	It("writes data", func() {
		err := naclBackend.WriteBlob("foo", bytes.NewBufferString("foobar"))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(backend["foo.nacl"]).ShouldNot(HaveLen(0))
	})

	It("reads data", func() {
		backend["foo.nacl"] = []byte{0x5d, 0x10, 0x39, 0x1c, 0x77, 0x2, 0xb, 0x26, 0x7e, 0xa6, 0x58, 0x52, 0xb9, 0x18, 0x55, 0x40, 0xb, 0x1, 0xd2, 0xc0, 0x40, 0xc9, 0xb3, 0xec, 0x27, 0x95, 0x9d, 0xf8, 0x17, 0x4b, 0xc7, 0xbb, 0xbb, 0x7, 0x31, 0x64, 0x66, 0xc9, 0xb9, 0xf8, 0x81, 0xdc, 0xef, 0xd, 0x6d, 0x56}
		rdr, err := naclBackend.ReadBlob("foo")
		Ω(err).ShouldNot(HaveOccurred())
		data, err := ioutil.ReadAll(rdr)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(data).Should(Equal([]byte("foobar")))
	})
})
