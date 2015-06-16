package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bargez/pktline"
	"github.com/codegangsta/cli"
	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/repos/local"
)

func main() {
	mainWithArgs(os.Args)
}

func mainWithArgs(args []string) {
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
	}
	app.Run(args)
}

func add(c *cli.Context) {
	if len(c.Args()) != 2 {
		fmt.Println("usage: git cr add <remote name> <url>")
		os.Exit(1)
	}
	remoteName := c.Args()[0]
	remoteURL := c.Args()[1]
	cmd := exec.Command("git", "remote", "add", remoteName, "ext::git cr run "+remoteURL+" %G")
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
	if len(c.Args()) != 1 {
		fmt.Println("don't run this manually, checkout git cr help :)")
		os.Exit(1)
	}

	repoURL := c.Args().First()

	// Load repo

	repo, err := local.NewLocalRepo(repoURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occured while initing the repo:\n%v\n", err)
	}

	// Handle request

	encoder := pktline.NewEncoder(os.Stdout)
	decoder := &pktlineDecoderWrapper{Decoder: pktline.NewDecoder(os.Stdin), Reader: os.Stdin}

	server := git.NewGitRequestHandler(encoder, decoder, repo)
	if err := server.ServeRequest(); err != nil {
		fmt.Fprintf(os.Stderr, "an error occured while serving git:\n%v\n", err)
	}
}
