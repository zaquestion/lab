package main

import (
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/gitlab"
)

func main() {
	gitlab.Init()
	cmd.Execute()
}
