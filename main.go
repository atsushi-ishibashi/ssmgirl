package main

import (
	"os"

	"github.com/atsushi-ishibashi/ssmgirl/cmd"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "awsconf",
			Usage: "set env var from ~/.aws/credentials in process",
		},
		cli.StringFlag{
			Name:  "awsregion",
			Usage: "set AWS_DEFAULT_REGION in process",
			Value: "ap-northeast-1",
		},
	}

	shellCommand := cmd.NewShellCommand()

	app.Commands = []cli.Command{
		shellCommand,
	}
	app.Run(os.Args)
}
