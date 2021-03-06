package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/bargez/pktline"
	"github.com/codegangsta/cli"
	"github.com/lucas-clemente/git-cr/backends/local"
	"github.com/lucas-clemente/git-cr/crypto/nacl"
	"github.com/lucas-clemente/git-cr/git/repo"
	"github.com/lucas-clemente/git-cr/git/handler"
)

func main() {
	app := cli.NewApp()
	app.Name = "git cr"
	app.Usage = "Encrypted git remote"
	app.Version = "0.1.0"
	app.Commands = []cli.Command{
		{
			Name:   "add",
			Usage:  "Setup a crypto remote in the current repo",
			Action: add,
		},
		{
			Name:   "run",
			Usage:  "Run the git server (should not be called manually)",
			Action: run,
		},
		{
			Name:   "clone",
			Usage:  "Clone from a crypto remote",
			Action: clone,
		},
	}
	app.Run(os.Args)
}

func add(c *cli.Context) {
	if len(c.Args()) != 3 {
		fmt.Println("usage: git cr add <remote name> <url> <encryption settings>")
		os.Exit(1)
	}
	remoteName := c.Args()[0]
	remoteURL := c.Args()[1]
	encryptionSettings := c.Args()[2]
	cmd := exec.Command("git", "remote", "add", remoteName, buildRemote(remoteURL, encryptionSettings))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git errored: %v\n%s", err, out)
		os.Exit(1)
	}
}

type pktlineDecoderWrapper struct {
	*pktline.Decoder
	io.Reader
}

func run(c *cli.Context) {
	if len(c.Args()) != 2 {
		fmt.Println("don't run this manually, checkout git cr help :)")
		os.Exit(1)
	}

	repoURLString := c.Args()[0]
	encryptionSettings := c.Args()[1]

	repoURL, err := url.Parse(repoURLString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occured while parsing the URL:\n%v\n", err)
		os.Exit(1)
	}

	// Load repo

	backend, err := local.NewLocalBackend(repoURL.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occured while initing the repo:\n%v\n", err)
		os.Exit(1)
	}

	// Wrap in encryption

	if encryptionSettings == "none" {
		// Do nothing
	} else if strings.HasPrefix(encryptionSettings, "nacl:") {
		secretB64 := strings.TrimPrefix(encryptionSettings, "nacl:")
		secret, err := base64.StdEncoding.DecodeString(secretB64)
		if err != nil || len(secret) != 32 {
			fmt.Fprintf(os.Stderr, "the nacl secret should be 32 bytes in base64")
			os.Exit(1)
		}

		secretArray := [32]byte{}
		copy(secretArray[:], secret)
		backend = nacl.NewNaClBackend(backend, secretArray)
	} else {
		fmt.Fprintf(os.Stderr, "the encryption settings are invalid")
		os.Exit(1)
	}

	// Setup repo

	repo := repo.NewJSONRepo(backend)

	// Handle request

	encoder := pktline.NewEncoder(os.Stdout)
	decoder := &pktlineDecoderWrapper{Decoder: pktline.NewDecoder(os.Stdin), Reader: os.Stdin}

	server := handler.NewGitRequestHandler(encoder, decoder, repo)
	if err := server.ServeRequest(); err != nil {
		fmt.Fprintf(os.Stderr, "an error occured while serving git:\n%v\n", err)
	}
}

func clone(c *cli.Context) {
	if len(c.Args()) < 2 {
		fmt.Println("usage: git cr clone <url> <encryption settings> [destination]")
		os.Exit(1)
	}
	remoteURL := c.Args()[0]
	encryptionSettings := c.Args()[1]

	cloneArgs := []string{"clone", buildRemote(remoteURL, encryptionSettings)}
	cloneArgs = append(cloneArgs, c.Args()[2:]...)
	cmd := exec.Command("git", cloneArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git errored: %v\n%s", err, out)
		os.Exit(1)
	}
}

func buildRemote(url, encryptionSettings string) string {
	return "ext::git cr %G run " + url + " " + encryptionSettings
}
