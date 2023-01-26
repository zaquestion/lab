package main

import (
	"os"

	"github.com/rsteube/carapace"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func main() {
	cmd.Version = version
	initSkipped := skipInit()
	if !initSkipped {
		h, u, t, ca, skipVerify := config.LoadMainConfig()

		if ca != "" {
			lab.InitWithCustomCA(h, u, t, ca)
		} else {
			lab.Init(h, u, t, skipVerify)
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
