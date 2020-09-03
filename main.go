package main

import (
	"log"
	"os"

	"github.com/rsteube/carapace"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Version = version
	if !skipInit() {
		h, u, t, ca, skipVerify := config.LoadConfig()

		if ca != "" {
			lab.InitWithCustomCA(h, u, t, ca)
		} else {
			lab.Init(h, u, t, skipVerify)
		}
	}
	cmd.Execute()
}

func skipInit() bool {
	if len(os.Args) <= 1 {
		return false
	}
	switch os.Args[1] {
	case "completion":
		return true
	case "_carapace":
		return !carapace.IsCallback()
	default:
		return false
	}
}
