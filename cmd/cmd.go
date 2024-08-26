package cmd

import "github.com/urfave/cli"

func Commands() []cli.Command {
	command := cli.Command{
		Name:   "action",
		Usage:  "run the action sidecar",
		Action: run,
	}

	return []cli.Command{command}
}
