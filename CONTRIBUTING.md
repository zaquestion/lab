
## Project Goals

*lab* is fundamentally a workflow tool; we don't add features just to cover the API.
Instead, we add them to support and improve our cli workflows, which we want to flow seamlessly and feel intuitive and natural.

## Overview of Tests

*lab* runs integration tests in addition to unit tests. The integration tests run against [gitlab.com](https://gitlab.com). We are willing to trade some test autonomy and speed in exchange for 100% guarantees that features work against a live GitLab API. Integration tests are largely identified as tests which execute the `./lab.test` binary. There are two primary projects used for these integration tests: [zaquestion/test](https://gitlab.com/zaquestion/test) and [lab-testing/test](https://gitlab.com/lab-testing/test).

## Setup and Prerequestites

**New to Go?** Check out the Go docs on [How to Write Go Code](https://golang.org/doc/code.html) for some background on Go coding conventions, many of which are used by *lab*.

To run the *lab* tests, you will need:
1. `go` and `git` must be installed (optionally `make`)
2. A gitlab.com account configured with an [SSH key](https://docs.gitlab.com/ce/ssh/README.html#adding-an-ssh-key-to-your-gitlab-account). If you can push and pull from gitlab.com remotes, you're probably all set.
3. The `GOPATH` environment variable needs to be explicitly set. (eg `export GOPATH=$(go env GOPATH)`)
4. Add `$GOPATH/bin` to your `$PATH`.
5. The `GO111MODULE` environment variable needs to be set to `on`. (eg `export GO111MODULE=on`)
6. The tests assume that the lab source repo is located in `$GOPATH/src/github.com/zaquestion/lab`

## Running Tests
Tests can be run via `make test`:

```sh
$ cd $GOPATH/src/github.com/zaquestion/lab

# run all tests
$ make test

# run only tests matching "pattern"
$ make test run=pattern
```

or with `go test`:

```sh
$ cd $GOPATH/src/github.com/zaquestion/lab

$ GO111MODULE=on go test ./cmd ./internal/...
```
