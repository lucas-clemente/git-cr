package merger_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/git/merger"
	"github.com/lucas-clemente/git-cr/repos/fixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPackfileMerger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Packfile Merger Suite")
}

var _ = Describe("PackfileMerger", func() {
	var (
		tempDir        string
		packfileMerger git.Repo
		repo           *fixture.FixtureRepo
	)

	BeforeEach(func() {
		var err error

		tempDir, err = ioutil.TempDir("", "io.clemente.git-cr.test.pack")
		Ω(err).ShouldNot(HaveOccurred())

		repo = fixture.NewFixtureRepo()
		repo.SaveNewRevisionB64(git.Revision{}, "UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=")
		repo.SaveNewRevisionB64(git.Revision{}, "UEFDSwAAAAIAAAADlgx4nJXLSwrCMBRG4XlWkbkgSe5NbgpS3Eoef1QwtrQRXL51CU7O4MA3NkDnmqgFT0CSBhIGI0RhmeBCCb5Mk2cbWa1pw2voFjmbKiQ+l2xDrU7YER8oNSuUgNxKq0Gl97gvmx7Yh778esUn9fWJc1n6rC0TG0suOn0yzhh13P4YA38Q1feb+gIlsDr0M3icS0qsAgACZQE+rwF4nDM0MDAzMVFIy89nsJ9qkZYUaGwfv1Tygdym9MuFp+ZUAACUGAuBskz7fFz81Do1iG8hcUrj/ncK63Q=")

		packfileMerger = merger.NewMerger(repo)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("reads original packfiles", func() {
		r, err := packfileMerger.ReadPackfile(0, 1)
		Ω(err).ShouldNot(HaveOccurred())
		d, err := ioutil.ReadAll(r)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(d).Should(Equal(repo.Packfiles[0]))
	})

	It("reads merged packfiles", func() {
		r, err := packfileMerger.ReadPackfile(0, 2)
		Ω(err).ShouldNot(HaveOccurred())
		pack, err := ioutil.ReadAll(r)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(pack[0:4]).Should(Equal([]byte("PACK")))
		Ω(pack[4:8]).Should(Equal([]byte{0, 0, 0, 2}))
		Ω(pack[8:12]).Should(Equal([]byte{0, 0, 0, 6}))

		err = ioutil.WriteFile(tempDir+"/pack.pack", pack, 0644)
		Ω(err).ShouldNot(HaveOccurred())

		err = exec.Command("git", "index-pack", "--strict", tempDir+"/pack.pack").Run()
		Ω(err).ShouldNot(HaveOccurred())

		out, err := exec.Command("git", "verify-pack", "-v", tempDir+"/pack.pack").CombinedOutput()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(out)).Should(ContainSubstring("f84b0d7375bcb16dd2742344e6af173aeebfcfd6"))
		Ω(string(out)).Should(ContainSubstring("5716ca5987cbf97d6bb54920bea6adde242d87e6"))
		Ω(string(out)).Should(ContainSubstring("6a09c59ce8eb1b5b4f89450103e67ff9b3a3b1ae"))
		Ω(string(out)).Should(ContainSubstring("1a6d946069d483225913cf3b8ba8eae4c894c322"))
		Ω(string(out)).Should(ContainSubstring("3f9538666251333f5fa519e01eb267d371ca9c78"))
		Ω(string(out)).Should(ContainSubstring("bda3f653eea7fe374e4e687479e26c65c9954184"))
	})
})
