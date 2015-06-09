package git_test

import (
	"errors"

	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type sampleDecoder struct {
	data [][]byte
}

func (d *sampleDecoder) Decode(b *[]byte) error {
	if len(d.data) == 0 {
		return errors.New("EOF")
	}
	*b = d.data[0]
	d.data = d.data[1:]
	return nil
}

func (d *sampleDecoder) setData(data ...[]byte) {
	d.data = data
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
			decoder.setData([]byte(""))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack "))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo"))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000"))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo\000"))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo\000host="))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000host=foo"))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000host="))
			Ω(handler.ParseHandshake()).Should(Equal(git.ErrorInvalidHandshake))
		})

		It("gets repo and host", func() {
			decoder.setData([]byte("git-upload-pack foo\000host=bar"))
			Ω(handler.ParseHandshake()).ShouldNot(HaveOccurred())
			Ω(handler.Host).Should(Equal("bar"))
			Ω(handler.Repo).Should(Equal("foo"))
		})
	})

	Context("sending refs", func() {
		It("sends empty list", func() {
			Ω(handler.SendRefs([]git.Ref{})).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(1))
			Ω(encoder.data[0]).Should(BeNil())
		})

		It("sends reflist", func() {
			refs := []git.Ref{git.Ref{Name: "foo", Sha1: "bar"}}
			Ω(handler.SendRefs(refs)).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("bar foo")))
			Ω(encoder.data[1]).Should(BeNil())
		})
	})

	Context("reading client wants", func() {
		It("receives wants", func() {
			decoder.setData(
				[]byte("want 30f79bec32243c31dd91a05c0ad7b80f1e301aea"),
				[]byte("want f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"),
				nil,
			)
			wants, err := handler.ReceiveClientWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(2))
			Ω(wants[0]).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
			Ω(wants[1]).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles client capabilities", func() {
			decoder.setData(
				[]byte("want 30f79bec32243c31dd91a05c0ad7b80f1e301aea\000foobar"),
				nil,
			)
			wants, err := handler.ReceiveClientWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(1))
			Ω(wants[0]).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
		})
	})
})
