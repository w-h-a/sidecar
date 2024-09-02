package cmd

import "github.com/urfave/cli"

func Commands() []cli.Command {
	command := cli.Command{
		Name:   "sidecar",
		Usage:  "run the sidecar",
		Action: run,
	}

	return []cli.Command{command}
}
