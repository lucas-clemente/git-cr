package git_test

import (
	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type sampleDecoder struct {
	data []byte
}

func (d *sampleDecoder) Decode(b *[]byte) error {
	*b = d.data
	return nil
}

type sampleEncoder struct {
	data [][]byte
}

func (d *sampleEncoder) Encode(b []byte) error {
	d.data = append(d.data, b)
	return nil
}

var _ = Describe("upload-pack", func() {
	var (
		decoder *sampleDecoder
		encoder *sampleEncoder
		handler *git.UploadPackHandler
	)

	BeforeEach(func() {
		decoder = &sampleDecoder{}
		encoder = &sampleEncoder{data: [][]byte{}}
		handler = git.NewUploadPackHandler(encoder, decoder)
	})

	Context("decoding client handshake", func() {
		It("errors on invalid handshake", func() {
			decoder.data = []byte("")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack ")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack foo")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack \000")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack foo\000")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack foo\000host=")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack \000host=foo")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.data = []byte("git-upload-pack \000host=")
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
		})

		It("gets repo and host", func() {
			decoder.data = []byte("git-upload-pack foo\000host=bar")
			Ω(handler.ParseHandshake()).ShouldNot(HaveOccurred())
			Ω(handler.Host).Should(Equal("bar"))
			Ω(handler.Repo).Should(Equal("foo"))
		})
	})

	Context("sending refs", func() {

	})
})
