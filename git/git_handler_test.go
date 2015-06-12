package git_test

import (
	"bytes"
	"io"
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
		return io.EOF
	}
	*b = d.data[0]
	d.data = d.data[1:]
	return nil
}

func (d *sampleDecoder) Read(p []byte) (int, error) {
	panic("not implemented")
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

func (*sampleBackend) GetRefs() ([]git.Ref, error) {
	panic("not implemented")
}

func (*sampleBackend) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	panic("not implemented")
}

func (*sampleBackend) UpdateRef(update git.RefUpdate) error {
	panic("not implemented")
}

func (*sampleBackend) WritePackfile(from, to string, r io.Reader) error {
	panic("not implemented")
}

var _ = Describe("git server", func() {
	var (
		decoder *sampleDecoder
		encoder *sampleEncoder
		backend *sampleBackend
		handler *git.GitRequestHandler
	)

	BeforeEach(func() {
		decoder = &sampleDecoder{}
		encoder = &sampleEncoder{data: [][]byte{}}
		backend = &sampleBackend{deltas: []*sampleDelta{}}
		handler = git.NewGitRequestHandler(encoder, decoder, backend)
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
		It("sends reflist for pull", func() {
			refs := []git.Ref{git.Ref{Name: "foo", Sha1: "bar"}}
			Ω(handler.SendRefs(refs, git.GitPull)).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("bar foo\000multi_ack_detailed side-band-64k thin-pack")))
			Ω(encoder.data[1]).Should(BeNil())
		})

		It("sends reflist for push", func() {
			refs := []git.Ref{git.Ref{Name: "foo", Sha1: "bar"}}
			Ω(handler.SendRefs(refs, git.GitPush)).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("bar foo\000delete-refs ofs-delta")))
			Ω(encoder.data[1]).Should(BeNil())
		})
	})

	Context("reading pull wants", func() {
		It("receives wants", func() {
			decoder.setData(
				[]byte("want 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("want f1d2d2f924e986ac86fdf7b36c94bcdf32beec15\n"),
				nil,
			)
			wants, err := handler.ReceivePullWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(2))
			Ω(wants[0]).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
			Ω(wants[1]).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles client capabilities", func() {
			decoder.setData(
				[]byte("want 30f79bec32243c31dd91a05c0ad7b80f1e301aea foobar\n"),
				nil,
			)
			wants, err := handler.ReceivePullWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(1))
			Ω(wants[0]).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
		})
	})

	Context("negotiating packfiles", func() {
		It("handles full deltas", func() {
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("done\n"),
			)
			wants := []string{"f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(1))
			Ω(encoder.data[0]).Should(Equal([]byte("NAK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal(""))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles intermediate flushes", func() {
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				nil,
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("done\n"),
			)
			wants := []string{"f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("NAK")))
			Ω(encoder.data[1]).Should(Equal([]byte("NAK")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal(""))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles single have with delta", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "30f79bec32243c31dd91a05c0ad7b80f1e301aea", to: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"},
			}
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea"),
				[]byte("done"),
			)
			wants := []string{"f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK 30f79bec32243c31dd91a05c0ad7b80f1e301aea ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK 30f79bec32243c31dd91a05c0ad7b80f1e301aea")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles single have with delta and followup haves", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "30f79bec32243c31dd91a05c0ad7b80f1e301aea", to: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"},
			}
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea"),
				[]byte("have e242ed3bffccdf271b7fbaf34ed72d089537b42f"),
				[]byte("done"),
			)
			wants := []string{"f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK 30f79bec32243c31dd91a05c0ad7b80f1e301aea ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK e242ed3bffccdf271b7fbaf34ed72d089537b42f ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("ACK e242ed3bffccdf271b7fbaf34ed72d089537b42f")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles single have with delta and irrelevant haves", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "30f79bec32243c31dd91a05c0ad7b80f1e301aea", to: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"},
			}
			decoder.setData(
				[]byte("have e242ed3bffccdf271b7fbaf34ed72d089537b42f"),
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea"),
				[]byte("done"),
			)
			wants := []string{"f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK 30f79bec32243c31dd91a05c0ad7b80f1e301aea ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK 30f79bec32243c31dd91a05c0ad7b80f1e301aea")))
			Ω(deltas).Should(HaveLen(1))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"))
		})

		It("handles multiple wants", func() {
			backend.deltas = []*sampleDelta{
				&sampleDelta{from: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1", to: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2"},
				&sampleDelta{from: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1", to: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2"},
			}
			decoder.setData(
				[]byte("have aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"),
				[]byte("have bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1"),
				[]byte("done"),
			)
			wants := []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2"}
			deltas, err := handler.NegotiatePullPackfile(wants)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1 common")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1 ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("ACK bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1")))
			Ω(deltas).Should(HaveLen(2))
			Ω(deltas[0].(*sampleDelta).from).Should(Equal("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"))
			Ω(deltas[0].(*sampleDelta).to).Should(Equal("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2"))
			Ω(deltas[1].(*sampleDelta).from).Should(Equal("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb1"))
			Ω(deltas[1].(*sampleDelta).to).Should(Equal("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2"))
		})
	})

	Context("sending packfiles", func() {
		It("sends short packfiles", func() {
			pack := bytes.NewBufferString("foobar")
			err := handler.SendPackfile(pack)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("\001foobar")))
			Ω(encoder.data[1]).Should(BeNil())
		})

		It("sends long packfiles", func() {
			data := make([]byte, 65519+1)
			src := rand.NewSource(42)
			for i := range data {
				data[i] = byte(src.Int63())
			}
			err := handler.SendPackfile(bytes.NewBuffer(data))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0][0]).Should(Equal(byte(1)))
			Ω(encoder.data[0][1:]).Should(HaveLen(65519))
			Ω(bytes.Equal(encoder.data[0][1:], data[0:65519])).Should(BeTrue())
			Ω(encoder.data[1][0]).Should(Equal(byte(1)))
			Ω(encoder.data[1][1]).Should(Equal(data[65519]))
			Ω(encoder.data[2]).Should(BeNil())
		})
	})

	Context("receiving push refs", func() {
		It("receives creates", func() {
			decoder.setData([]byte("0000000000000000000000000000000000000000 f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 refs/heads/master\n"), nil)
			refs, err := handler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]git.RefUpdate{git.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives with trailing NUL", func() {
			decoder.setData([]byte("0000000000000000000000000000000000000000 f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 refs/heads/master\000"), nil)
			refs, err := handler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]git.RefUpdate{git.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives updates", func() {
			decoder.setData([]byte("30f79bec32243c31dd91a05c0ad7b80f1e301aea f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 refs/heads/master\n"), nil)
			refs, err := handler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]git.RefUpdate{git.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "30f79bec32243c31dd91a05c0ad7b80f1e301aea",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives deletes", func() {
			decoder.setData([]byte("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 0000000000000000000000000000000000000000 refs/heads/master\n"), nil)
			refs, err := handler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]git.RefUpdate{git.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				NewID: "",
			}}))
		})
	})
})
