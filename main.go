package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/rsteube/carapace"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cmd.Version = version
	initSkipped := skipInit()
	if !initSkipped {
		h, u, t, ca, skipVerify := config.LoadMainConfig()

		if ca != "" {
			if err := lab.InitWithCustomCA(ctx, h, u, t, ca); err != nil {
				log.Fatal(err)
			}
		} else {
			lab.Init(ctx, h, u, t, skipVerify)
		}
	}
	cmd.Execute(initSkipped)
}

func skipInit() bool {
	if carapace.IsCallback() {
		return true
	}

	nArgs := len(os.Args)
	if nArgs <= 1 {
		return false
	}
	switch os.Args[nArgs-1] {
	case "-h", "--help":
		return true
	}
	switch os.Args[1] {
	case "-v", "--version", "version":
		return true
	case "-h", "--help", "help":
		return true
	case "completion":
		return true
	default:
		return false
	}
}
