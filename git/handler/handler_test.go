package handler_test

import (
	"bytes"
	"io"
	"math/rand"

	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/git/handler"
	"github.com/lucas-clemente/git-cr/repos/fixture"

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

var _ = Describe("git server", func() {
	var (
		decoder    *sampleDecoder
		encoder    *sampleEncoder
		repo       *fixture.FixtureRepo
		gitHandler *handler.GitRequestHandler
	)

	BeforeEach(func() {
		decoder = &sampleDecoder{}
		encoder = &sampleEncoder{data: [][]byte{}}
		repo = fixture.NewFixtureRepo()
		gitHandler = handler.NewGitRequestHandler(encoder, decoder, repo)
	})

	Context("decoding client handshake", func() {
		It("handles pulls", func() {
			decoder.setData([]byte("git-upload-pack foo\000host=bar"))
			op, err := gitHandler.ReceiveHandshake()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(op).Should(Equal(handler.GitPull))
		})

		It("handles pushes", func() {
			decoder.setData([]byte("git-receive-pack foo\000host=bar"))
			op, err := gitHandler.ReceiveHandshake()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(op).Should(Equal(handler.GitPush))
		})
	})

	Context("sending refs", func() {
		It("sends reflist for pull", func() {
			refs := map[string]string{"HEAD": "bar", "foo": "bar"}
			Ω(gitHandler.SendRefs(refs, handler.GitPull)).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("bar HEAD\000multi_ack_detailed side-band-64k thin-pack")))
			Ω(encoder.data[1]).Should(Equal([]byte("bar foo")))
			Ω(encoder.data[2]).Should(BeNil())
		})

		It("sends reflist for push", func() {
			refs := map[string]string{"HEAD": "bar", "foo": "bar"}
			Ω(gitHandler.SendRefs(refs, handler.GitPush)).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("bar HEAD\000delete-refs ofs-delta")))
			Ω(encoder.data[1]).Should(Equal([]byte("bar foo")))
			Ω(encoder.data[2]).Should(BeNil())
		})
	})

	Context("reading pull wants", func() {
		It("receives wants", func() {
			decoder.setData(
				[]byte("want 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("want f1d2d2f924e986ac86fdf7b36c94bcdf32beec15\n"),
				nil,
			)
			wants, err := gitHandler.ReceivePullWants()
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
			wants, err := gitHandler.ReceivePullWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(1))
			Ω(wants[0]).Should(Equal("30f79bec32243c31dd91a05c0ad7b80f1e301aea"))
		})

		It("receives empty wants", func() {
			decoder.setData(nil)
			wants, err := gitHandler.ReceivePullWants()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(wants).Should(HaveLen(0))
		})
	})

	Context("negotiating packfiles", func() {
		It("handles full deltas", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				},
			}
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("done\n"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(1))
			Ω(encoder.data[0]).Should(Equal([]byte("NAK")))
			Ω(i).Should(Equal(0))
		})

		It("handles intermediate flushes", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				},
			}
			decoder.setData(
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				nil,
				[]byte("have 30f79bec32243c31dd91a05c0ad7b80f1e301aea\n"),
				[]byte("done\n"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("NAK")))
			Ω(encoder.data[1]).Should(Equal([]byte("NAK")))
			Ω(i).Should(Equal(0))
		})

		It("handles single have with delta", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				},
			}
			decoder.setData(
				[]byte("have f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"),
				[]byte("done"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")))
			Ω(i).Should(Equal(0))
		})

		It("handles single have with delta and followup haves", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				},
			}
			decoder.setData(
				[]byte("have f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"),
				[]byte("have e242ed3bffccdf271b7fbaf34ed72d089537b42f"),
				[]byte("done"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK e242ed3bffccdf271b7fbaf34ed72d089537b42f ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("ACK e242ed3bffccdf271b7fbaf34ed72d089537b42f")))
			Ω(i).Should(Equal(0))
		})

		It("handles single have with multiple revisions", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "103ad77dc08d41c0b7490967903ac276c2b5cfce",
				},
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				},
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
					"refs/heads/foobar": "30f79bec32243c31dd91a05c0ad7b80f1e301aea",
				},
			}
			decoder.setData(
				[]byte("have f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"),
				[]byte("done"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(2))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 ready")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")))
			Ω(i).Should(Equal(1))
		})

		It("handles multiple haves with multiple revisions", func() {
			revisions := []git.Revision{
				git.Revision{
					"refs/heads/master": "103ad77dc08d41c0b7490967903ac276c2b5cfce",
				},
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
					"refs/heads/foobar": "d54852cea1ae42ee83c244b23190b03245b62a27",
				},
				git.Revision{
					"refs/heads/master": "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
					"refs/heads/foobar": "30f79bec32243c31dd91a05c0ad7b80f1e301aea",
				},
			}
			decoder.setData(
				[]byte("have f1d2d2f924e986ac86fdf7b36c94bcdf32beec15"),
				[]byte("have d54852cea1ae42ee83c244b23190b03245b62a27"),
				[]byte("done"),
			)
			i, err := gitHandler.NegotiatePullPackfile(revisions)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(encoder.data).Should(HaveLen(3))
			Ω(encoder.data[0]).Should(Equal([]byte("ACK f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 common")))
			Ω(encoder.data[1]).Should(Equal([]byte("ACK d54852cea1ae42ee83c244b23190b03245b62a27 ready")))
			Ω(encoder.data[2]).Should(Equal([]byte("ACK d54852cea1ae42ee83c244b23190b03245b62a27")))
			Ω(i).Should(Equal(1))
		})
	})

	Context("sending packfiles", func() {
		It("sends short packfiles", func() {
			pack := bytes.NewBufferString("foobar")
			err := gitHandler.SendPackfile(pack)
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
			err := gitHandler.SendPackfile(bytes.NewBuffer(data))
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
			refs, err := gitHandler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]handler.RefUpdate{handler.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives with trailing NUL", func() {
			decoder.setData([]byte("0000000000000000000000000000000000000000 f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 refs/heads/master\000"), nil)
			refs, err := gitHandler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]handler.RefUpdate{handler.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives updates", func() {
			decoder.setData([]byte("30f79bec32243c31dd91a05c0ad7b80f1e301aea f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 refs/heads/master\n"), nil)
			refs, err := gitHandler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]handler.RefUpdate{handler.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "30f79bec32243c31dd91a05c0ad7b80f1e301aea",
				NewID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
			}}))
		})

		It("receives deletes", func() {
			decoder.setData([]byte("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 0000000000000000000000000000000000000000 refs/heads/master\n"), nil)
			refs, err := gitHandler.ReceivePushRefs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(refs).Should(Equal([]handler.RefUpdate{handler.RefUpdate{
				Name:  "refs/heads/master",
				OldID: "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15",
				NewID: "",
			}}))
		})
	})
})
