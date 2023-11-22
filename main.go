package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	var serveOpts []tf6server.ServeOpt
	if *debugFlag {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	serverFactory, err := provider.NewMuxedProviderServer(context.Background(), version)
	if err != nil {
		log.Fatal(err)
	}

	err = tf6server.Serve(
		"registry.terraform.io/gitlabhq/gitlab",
		serverFactory,
		serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
