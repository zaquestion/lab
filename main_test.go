package main

// This file is mandatory as otherwise the lab.test binary is not generated correctly.
import (
	"flag"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

// Test started when the test binary is started. Only calls main.
func TestLab(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	flag.Set("test.coverprofile", "../coverage-"+strconv.Itoa(int(rand.Uint64()))+".out")
	main()
}
