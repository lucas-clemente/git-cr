package main

import (
	"encoding/base64"
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

func configGit(dir string) {
	cmd := exec.Command("git", "config", "user.name", "test")
	cmd.Dir = dir
	err := cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	err = cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
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
		Ω(output).Should(ContainSubstring("origin\text::git cr %G run " + remoteDir))
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

		configGit(workingDir)

		cmd = exec.Command("git", "commit", "-m", "test")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())

		cmd = exec.Command("git", "push", "origin", "master")
		err = cmd.Run()
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("clones", func() {
		Ω(os.Chdir(workingDir)).ShouldNot(HaveOccurred())

		err := ioutil.WriteFile(remoteDir+"/refs.json", []byte(`{"HEAD":"f84b0d7375bcb16dd2742344e6af173aeebfcfd6","refs/heads/master":"f84b0d7375bcb16dd2742344e6af173aeebfcfd6"}`), 0644)
		Ω(err).ShouldNot(HaveOccurred())

		pack, err := base64.StdEncoding.DecodeString("UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=")
		Ω(err).ShouldNot(HaveOccurred())
		err = ioutil.WriteFile(remoteDir+"/_f84b0d7375bcb16dd2742344e6af173aeebfcfd6.pack", pack, 0644)
		Ω(err).ShouldNot(HaveOccurred())

		mainWithArgs([]string{"", "clone", remoteDir, "."})

		data, err := ioutil.ReadFile("foo")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(data).Should(Equal([]byte("bar\n")))
	})
})
