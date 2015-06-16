package main

import (
	"io/ioutil"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGitCr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = Describe("Main", func() {
	var (
		tmpDir  string
		tmpDir2 string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())
		tmpDir2, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		os.RemoveAll(tmpDir2)
	})

	It("adds remotes", func() {
		Ω(os.Chdir(tmpDir)).ShouldNot(HaveOccurred())
		cmd := exec.Command("git", "init")
		err := cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		mainWithArgs([]string{"", "add", "origin", tmpDir2})

		cmd = exec.Command("git", "remote", "-v")
		output, err := cmd.CombinedOutput()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(output).Should(ContainSubstring("origin\text::git cr run " + tmpDir2))
	})
})
