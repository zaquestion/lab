
## Project Goals

lab is fundamentally a workflow tool; we don't add features just to cover the API.
Instead, we add them to support and improve our cli workflows, which we want to flow seamlessly and feel intuitive and natural.

## Overview of Tests

lab runs integration tests in addition to unit tests. The integration tests run against gitlab.com. We are willing to trade some test autonomy and speed in exchange for 100% guarantees that features work against a live GitLab API. Integration tests are largely identified as tests which execute the `./lab_bin` binary. There are two primary projects used for these integration tests: [zaquestion/test](https://gitlab.com/zaquestion/test) and [lab-testing/test](https://gitlab.com/lab-testing/test).

## Setup and Prerequestites

**New to Go?** Check out the Go docs on [How to Write Go Code](https://golang.org/doc/code.html) for some background on Go coding conventions, many of which are used by lab.

To run the lab tests, you will need:
1. A gitlab.com account configured with an [SSH key](https://docs.gitlab.com/ce/ssh/README.html#adding-an-ssh-key-to-your-gitlab-account). If you can push and pull from gitlab.com remotes, you're probably all set.
2. The `GOPATH` environment variable needs to be explicitly set. (eg `export GOPATH=$(go env GOPATH)`)
3. Add `$GOPATH/bin` to your `$PATH`.
3. The `GO111MODULE` environment variable needs to be set to `on`. (eg `export GO111MODULE=on`)
4. The tests assume that the lab source repo is located in `$GOPATH/src/github.com/zaquestion/lab`
4. It should go without saying, but you'll also need `go`, `git` and `make` installed.

## Running Tests
In order to setup the integration test data, the tests must be run via `make test`:

```sh
$ cd $GOPATH/src/github.com/zaquestion/lab

# run all tests
$ make test

# run only tests matching "pattern"
$ make test run=pattern
```
