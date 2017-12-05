package main

import (
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/gitlab"
)

var version = "master"

func main() {
	cmd.Version = version
	gitlab.Init()
	cmd.Execute()
}
