package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()

	app.Name = "Slack Advanced Exporter"
	app.Usage = "A tool to augment official Slack data exports with additional data that Slack does not include by default."
	app.Version = "0.3.1"

	var inputArchive string
	var outputArchive string
	var slackApiToken string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "input-archive, i",
			Value:       "",
			Usage:       "The path to the Slack export archive which you wish to augment.",
			Destination: &inputArchive,
		},
		cli.StringFlag{
			Name:        "output-archive, o",
			Value:       "",
			Usage:       "The path to which you would like the output archive to be written.",
			Destination: &outputArchive,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "fetch-attachments",
			Aliases: []string{"a"},
			Usage:   "Fetch all file attachments and add them to the output archive.",
			Action: func(c *cli.Context) error {
				return fetchAttachments(inputArchive, outputArchive, slackApiToken)
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "api-token",
					Usage:       "Slack API token. Can be obtained here: https://api.slack.com/docs/oauth-test-tokens",
					Destination: &slackApiToken,
				},
			},
		},
		{
			Name:    "fetch-emails",
			Aliases: nil,
			Usage:   "Fetch users' e-mail addresses using Slack API and them to the output archive.",
			Action: func(c *cli.Context) error {
				return fetchEmails(inputArchive, outputArchive, slackApiToken)
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "api-token",
					Usage:       "Slack API token. Can be obtained here: https://api.slack.com/docs/oauth-test-tokens",
					Destination: &slackApiToken,
				},
			},
		},
	}

	app.Run(os.Args)
}
