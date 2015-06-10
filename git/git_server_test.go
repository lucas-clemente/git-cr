package git_test

import (
	"bytes"
	"errors"
	"math/rand"

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

type sampleDelta struct {
	from string
	to   string
}

type sampleBackend struct {
	deltas []*sampleDelta
}

func (b *sampleBackend) FindDelta(from, to string) (git.Delta, error) {
	for _, d := range b.deltas {
		if d.from == from && d.to == to {
			return d, nil
		}
	}
	return nil, nil
}

func (b *sampleBackend) DeltaFromZero(to string) (git.Delta, error) {
	return &sampleDelta{from: "", to: to}, nil
}

var _ = Describe("upload-pack", func() {
	var (
		decoder *sampleDecoder
		encoder *sampleEncoder
		backend *sampleBackend
		handler *git.GitServer
	)

	BeforeEach(func() {
		decoder = &sampleDecoder{}
		encoder = &sampleEncoder{data: [][]byte{}}
		backend = &sampleBackend{deltas: []*sampleDelta{}}
		handler = git.NewGitServer(encoder, decoder, backend)
	})

	Context("decoding client handshake", func() {
		It("errors on invalid handshake", func() {
			decoder.setData([]byte(""))
			_, err := handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack "))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo"))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000"))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo\000"))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack foo\000host="))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000host=foo"))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			decoder.setData([]byte("git-upload-pack \000host="))
			_, err = handler.ReceiveHandshake()
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
		})

		It("gets repo and host", func() {
			decoder.setData([]byte("git-upload-pack foo\000host=bar"))
			op, err := handler.ReceiveHandshake()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(op).Should(Equal(git.GitPull))
			Ω(handler.Host).Should(Equal("bar"))
			Ω(handler.Repo).Should(Equal("foo"))
		})

		It("handles pushes", func() {
			decoder.setData([]byte("git-receive-pack foo\000host=bar"))
			op, err := handler.ReceiveHandshake()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(op).Should(Equal(git.GitPush))
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
			Ω(encoder.data[0]).Should(Equal([]byte("bar foo\000multi_ack_detailed side-band-64k thin-pack")))
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

	Context("negotiating packfiles", func() {
		It("handles full deltas", func() {
			decoder.setData(
				[]byte("have foobar"),
				[]byte("done"),
			)
			wants := []string{"another"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(1))
			Ω(encoder.data[0]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal(""))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("another"))
		})

		It("handles intermediate flushes", func() {
			decoder.setData(
				[]byte("have foobar"),
				nil,
				[]byte("have foobar"),
				[]byte("done"),
			)
			wants := []string{"another"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("NACK")))
			Ω(encoder.data[1]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal(""))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("another"))
		})

		It("handles single have with delta", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "foobar", to: "foobaz"},
			}
			decoder.setData(
				[]byte("have foobar"),
				[]byte("done"),
			)
			wants := []string{"foobaz"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK foobar ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("foobar"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("foobaz"))
		})

		It("handles single have with delta and followup haves", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "foobar", to: "foobaz"},
			}
			decoder.setData(
				[]byte("have foobar"),
				[]byte("have somethingelse"),
				[]byte("done"),
			)
			wants := []string{"foobaz"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK foobar ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK somethingelse ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("foobar"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("foobaz"))
		})

		It("handles single have with delta and irrelevant haves", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "foobar", to: "foobaz"},
			}
			decoder.setData(
				[]byte("have somethingelse"),
				[]byte("have foobar"),
				[]byte("done"),
			)
			wants := []string{"foobaz"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK foobar ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("foobar"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("foobaz"))
		})

		It("handles multiple wants", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "a1", to: "a2"},
				&sampleDelta{from: "b1", to: "b2"},
			}
			decoder.setData(
				[]byte("have a1"),
				[]byte("have b1"),
				[]byte("done"),
			)
			wants := []string{"a2", "b2"}
			deltas, err := handler.HandleClientHaves(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK a1 common")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK b1 ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("NACK")))
			Ω(deltas).Should(HaveLen(2))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("a1"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("a2"))
			Ω(deltas[1].(*sampleDelta).from).Should(Equal("b1"))
			Ω(deltas[1].(*sampleDelta).to).Should(Equal("b2"))
		})
	})

	Context("sending packfiles", func() {
		It("sends short packfiles", func() {
			pack := bytes.NewBufferString("foobar")
			err := handler.SendPackfile(pack)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(1))
			Ω(encoder.data[0]).Should(Equal([]byte("\001foobar")))
		})

		It("sends long packfiles", func() {
			data := make([]byte, 65519+1)
			src := rand.NewSource(42)
			for i := range data {
				data[i] = byte(src.Int63())
			}
			err := handler.SendPackfile(bytes.NewBuffer(data))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0][0]).Should(Equal(byte(1)))
			Ω(encoder.data[0][1:]).Should(HaveLen(65519))
			Ω(bytes.Equal(encoder.data[0][1:], data[0:65519])).Should(BeTrue())
			Ω(encoder.data[1][0]).Should(Equal(byte(1)))
			Ω(encoder.data[1][1]).Should(Equal(data[65519]))
		})
	})
})
