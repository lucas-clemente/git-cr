package git_test

import (
	"github.com/lucas-clemente/git-cr/git"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("upload-pack", func() {
	Context("decoding client handshake", func() {
		It("errors on invalid handshake", func() {
			_, _, err := git.HandshakePull([]byte(""))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack "))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack foo"))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack \000"))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack foo\000"))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack foo\000host="))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack \000host=foo"))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
			_, _, err = git.HandshakePull([]byte("git-upload-pack \000host="))
			Ω(err).Should(Equal(git.ErrorInvalidHandshake))
		})

		It("gets repo and host", func() {
			repo, host, err := git.HandshakePull([]byte("git-upload-pack foo\000host=bar"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(host).Should(Equal("bar"))
			Ω(repo).Should(Equal("foo"))
		})
	})
})
