package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/sidecar/cmd"
)

func main() {
	app := cli.NewApp()

	setup(app)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setup(app *cli.App) {
	app.Name = "cli"

	app.Usage = "run server"

	app.HideVersion = true

	app.Before = before

	app.Action = func(ctx *cli.Context) {}

	app.Commands = append(app.Commands, cmd.Commands()...)
}

func before(ctx *cli.Context) error {
	return nil
}
