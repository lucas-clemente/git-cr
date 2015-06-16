package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/codegangsta/cli"
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
	cmd := exec.Command("git", "remote", "add", remoteName, "ext::git cr run "+remoteURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git errored: %v\n%s", err, out)
		os.Exit(1)
	}
}

func run(c *cli.Context) {

}
