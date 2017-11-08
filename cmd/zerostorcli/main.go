package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli"
)

var (
	confFile string

	// CommitHash represents the Git commit hash at built time
	CommitHash string
	// BuildDate represents the date when this tool suite was built
	BuildDate string
)

func outputVersion() string {
	// Tool Version
	version := "Version: " + "1.1.0-alpha-8"

	// Build (Git) Commit Hash
	if CommitHash != "" {
		version += "\r\nBuild: " + CommitHash
		if BuildDate != "" {
			version += " " + BuildDate
		}
	}

	// Output version and runtime information
	return fmt.Sprintf("%s\r\nRuntime: %s %s\r\n",
		version,
		runtime.Version(), // Go Version
		runtime.GOOS,      // OS Name
	)
}

func main() {
	app := cli.NewApp()
	app.Version = outputVersion()
	app.Name = "0-stor cli"
	app.Usage = "Interact with 0-stors"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "conf",
			Value:       "config.yaml",
			Usage:       "path to the configuration file",
			Destination: &confFile,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "file",
			Usage: "Command to upload/download files",
			Subcommands: []cli.Command{
				{
					Name:  "upload",
					Usage: "upload a file. e.g: cli file upload myfile",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "key, k",
							Usage: "key to use to store the file, if empty use the name of the file as key",
						},
						cli.StringFlag{
							Name:  "reference, ref",
							Usage: "references for this file, split by comma for multiple values",
						},
					},
					Action: upload,
				},
				{
					Name:   "download",
					Usage:  "download a file. e.g: cli file download myfile",
					Action: download,
				},
				{
					Name:   "delete",
					Usage:  "delete a file. e.g: cli file delete myfile",
					Action: delete,
				},
				{
					Name:  "metadata",
					Usage: "print the metadata of a key",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "json",
							Usage: "print the metadata in JSON format",
						},
						cli.BoolFlag{
							Name:  "pretty",
							Usage: "print the metadata in a prettified JSON format",
						},
					},
					Action: metadata,
				},
				{
					Name:   "repair",
					Usage:  "Repair file",
					Action: repair,
				},
			},
		},
		{
			Name:  "namespace",
			Usage: "Manage namespaces",
			Subcommands: []cli.Command{
				{
					Name:   "create",
					Usage:  "Create a namespace. e.g: 'cli namespace create mynamespace'",
					Action: createNamespace,
				},

				{
					Name:   "delete",
					Usage:  "Delete a namespace. e.g: 'cli namespace delete mynamespace'",
					Action: deleteNamespace,
				},
				{
					Name:  "set-acl",
					Usage: "Set permission to a user",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "namespace",
							Usage: "Label of the namespace",
						},
						cli.StringFlag{
							Name:  "userid",
							Usage: "ItsYouOnline user's ID (email address)",
						},
						cli.BoolFlag{
							Name:  "read, r",
							Usage: "set read permission",
						},
						cli.BoolFlag{
							Name:  "write, w",
							Usage: "set write permission",
						},
						cli.BoolFlag{
							Name:  "delete, d",
							Usage: "set delete permission",
						},
						cli.BoolFlag{
							Name:  "admin, a",
							Usage: "set admin permission",
						},
					},
					Action: setACL,
				},
				{
					Name:  "get-acl",
					Usage: "Get the permission of a user",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "namespace",
							Usage: "Label of the namespace",
						},
						cli.StringFlag{
							Name:  "userid",
							Usage: "ItsYouOnline user's ID (email address)",
						},
					},
					Action: getACL,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
