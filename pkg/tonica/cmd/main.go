package cmd

import (
	"context"
	"log"
	"os"

	"github.com/tonica-go/tonica/pkg/tonica/cmd/docker_compose"
	"github.com/tonica-go/tonica/pkg/tonica/cmd/project"
	"github.com/tonica-go/tonica/pkg/tonica/cmd/proto_init"
	"github.com/tonica-go/tonica/pkg/tonica/cmd/wrap"
	"github.com/urfave/cli/v3"
)

func Run() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "init",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Value:    "app",
						Usage:    "Name for the project",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.String("name")
					err := project.CreateProject(name)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name: "proto",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Value:    "service",
						Usage:    "Name for proto file",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.String("name")
					err := proto_init.CreateProto(name)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name: "wrap",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "proto",
						Value:    "",
						Usage:    "path to proto file",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					path := cmd.String("proto")
					_, err := wrap.BuildGRPCGoFrServer(path)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name: "install",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := project.InstallGoDeps()
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "compose",
				Usage: "Generate docker-compose.yml with selected services",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := docker_compose.GenerateDockerCompose()
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
