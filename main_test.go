package main

import (
	"io/ioutil"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestGitCr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = Describe("Main", func() {
	var (
		workingDir  string
		remoteDir   string
		pathToGitCR string
	)

	BeforeSuite(func() {
		var err error
		pathToGitCR, err = gexec.Build("github.com/lucas-clemente/git-cr")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	BeforeEach(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())
		remoteDir, err = ioutil.TempDir("", "io.clemente.git-cr.test")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(workingDir)
		os.RemoveAll(remoteDir)
	})

	It("adds remotes", func() {
		Ω(os.Chdir(workingDir)).ShouldNot(HaveOccurred())
		cmd := exec.Command("git", "init")
		err := cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		mainWithArgs([]string{"", "add", "origin", remoteDir})

		cmd = exec.Command("git", "remote", "-v")
		output, err := cmd.CombinedOutput()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(output).Should(ContainSubstring("origin\text::git %G cr run " + remoteDir))
	})

	It("pushes updates", func() {
		Ω(os.Chdir(workingDir)).ShouldNot(HaveOccurred())
		cmd := exec.Command("git", "init")
		err := cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "remote", "add", "origin", "ext::"+pathToGitCR+" %G run "+remoteDir)
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		err = ioutil.WriteFile(workingDir+"/foo", []byte("foobar"), 0644)
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "add", "foo")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "config", "user.name", "test")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "commit", "-m", "test")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "push", "origin", "master")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())
	})
})
