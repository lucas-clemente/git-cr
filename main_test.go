package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestGitCr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

func runCommandInDir(dir, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	err := cmd.Run()
	Ω(err).ShouldNot(HaveOccurred())
}

func configGit(dir string) {
	runCommandInDir(dir, "git", "config", "user.name", "test")
	runCommandInDir(dir, "git", "config", "user.email", "test@example.com")
}

var _ = Describe("Main", func() {
	var (
		workingDir         string
		remoteDir          string
		pathToGitCR        string
		folderOfGitCR      string
		encryptionSettings string
	)

	BeforeSuite(func() {
		var err error
		pathToGitCR, err = gexec.Build("github.com/lucas-clemente/git-cr")
		Ω(err).ShouldNot(HaveOccurred())
		folderOfGitCR = filepath.Dir(pathToGitCR)
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

	remoteURL := func() string {
		return "ext::" + pathToGitCR + " %G run " + "file://" + remoteDir + " " + encryptionSettings
	}

	AfterEach(func() {
		os.RemoveAll(workingDir)
		os.RemoveAll(remoteDir)
	})

	sharedTests := func() {
		It("adds remotes", func() {
			cmd := exec.Command("git", "init", workingDir)
			err := cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			runCommandInDir(workingDir, pathToGitCR, "add", "origin", remoteDir, encryptionSettings)

			cmd = exec.Command("git", "remote", "-v")
			cmd.Dir = workingDir
			output, err := cmd.CombinedOutput()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(output).Should(ContainSubstring("origin\text::git cr %G run " + remoteDir + " " + encryptionSettings))
		})

		It("pushes updates", func() {
			runCommandInDir(workingDir, "git", "init")
			configGit(workingDir)

			runCommandInDir(workingDir, "git", "remote", "add", "origin", remoteURL())

			err := ioutil.WriteFile(workingDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())

			runCommandInDir(workingDir, "git", "add", "foo")
			runCommandInDir(workingDir, "git", "commit", "-m", "test")
			runCommandInDir(workingDir, "git", "push", "origin", "master")
		})

		It("force-pushes and clones", func() {
			runCommandInDir(workingDir, "git", "init")
			configGit(workingDir)

			err := ioutil.WriteFile(workingDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(workingDir, "git", "add", "foo")
			runCommandInDir(workingDir, "git", "commit", "-m", "test")

			err = ioutil.WriteFile(workingDir+"/bar", []byte("foobaz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(workingDir, "git", "add", "bar")
			runCommandInDir(workingDir, "git", "commit", "-m", "test2")
			runCommandInDir(workingDir, "git", "remote", "add", "origin", remoteURL())
			runCommandInDir(workingDir, "git", "push", "origin", "master")
			runCommandInDir(workingDir, "git", "reset", "--hard", "HEAD^")
			runCommandInDir(workingDir, "git", "push", "-f", "origin", "master")

			// Now try cloning

			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", remoteURL(), workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			contents, err := ioutil.ReadFile(workingDir2 + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("foobar")))
			_, err = ioutil.ReadFile(workingDir2 + "/bar")
			Ω(os.IsNotExist(err)).Should(BeTrue())
		})

		It("pushes multiple times and clones", func() {
			runCommandInDir(workingDir, "git", "init")
			configGit(workingDir)

			err := ioutil.WriteFile(workingDir+"/foo", []byte("foobar"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(workingDir, "git", "add", "foo")
			runCommandInDir(workingDir, "git", "commit", "-m", "test")
			runCommandInDir(workingDir, "git", "remote", "add", "origin", remoteURL())
			runCommandInDir(workingDir, "git", "push", "origin", "master")

			err = ioutil.WriteFile(workingDir+"/bar", []byte("foobaz"), 0644)
			Ω(err).ShouldNot(HaveOccurred())
			runCommandInDir(workingDir, "git", "add", "bar")
			runCommandInDir(workingDir, "git", "commit", "-m", "test2")
			runCommandInDir(workingDir, "git", "push", "origin", "master")

			// Now try cloning

			workingDir2, err := ioutil.TempDir("", "io.clemente.git-cr.test")
			Ω(err).ShouldNot(HaveOccurred())
			defer os.RemoveAll(workingDir2)

			cmd := exec.Command("git", "clone", remoteURL(), workingDir2)
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			contents, err := ioutil.ReadFile(workingDir2 + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("foobar")))
			contents, err = ioutil.ReadFile(workingDir2 + "/bar")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(contents).Should(Equal([]byte("foobaz")))
		})
	}

	Context("without encryption", func() {
		BeforeEach(func() {
			encryptionSettings = "none"
		})

		sharedTests()

		It("clones fixtures", func() {
			err := ioutil.WriteFile(remoteDir+"/refs.json", []byte(`{"HEAD":"f84b0d7375bcb16dd2742344e6af173aeebfcfd6","refs/heads/master":"f84b0d7375bcb16dd2742344e6af173aeebfcfd6"}`), 0644)
			Ω(err).ShouldNot(HaveOccurred())

			pack, err := base64.StdEncoding.DecodeString("UEFDSwAAAAIAAAADlwt4nJ3MQQrCMBBA0X1OMXtBJk7SdEBEcOslJmGCgaSFdnp/ET2By7f43zZVmAS5RC46a/Y55lBnDhE9kk6pVs4klL2ok8Ne6wbPo8gOj65DF1O49o/v5edzW2/gAxEnShzghBdEV9Yxmpn+V7u2NGvS4btxb5cEOSI0eJxLSiziAgADnQFArwF4nDM0MDAzMVFIy89nCBc7Fdl++mdt9lZPhX3L1t5T0W1/BgCtgg0ijmEEgEsIHYPJopDmNYTk3nR5stM=")
			Ω(err).ShouldNot(HaveOccurred())
			err = ioutil.WriteFile(remoteDir+"/_f84b0d7375bcb16dd2742344e6af173aeebfcfd6.pack", pack, 0644)
			Ω(err).ShouldNot(HaveOccurred())

			cmd := exec.Command(pathToGitCR, "clone", remoteDir, encryptionSettings, workingDir)
			// git-cr needs to be in the PATH to be discovered by the ext:: remote
			cmd.Env = []string{"PATH=" + folderOfGitCR + ":" + os.Getenv("PATH")}
			err = cmd.Run()
			Ω(err).ShouldNot(HaveOccurred())

			data, err := ioutil.ReadFile(workingDir + "/foo")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(Equal([]byte("bar\n")))
		})
	})

	Context("with nacl encryption", func() {
		BeforeEach(func() {
			encryptionSettings = "nacl:MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
		})

		sharedTests()
	})
})
