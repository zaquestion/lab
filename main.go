package main

import (
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/gitlab"
	"log"
)

var version = "master"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Version = version
	gitlab.Init()
	cmd.Execute()
}
