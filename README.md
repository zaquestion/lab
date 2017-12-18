# git + lab = gitlab [![Build Status](https://travis-ci.org/zaquestion/lab.svg?branch=master)](https://travis-ci.org/zaquestion/lab) [![Go Report Card](https://goreportcard.com/badge/github.com/zaquestion/lab)](https://goreportcard.com/report/github.com/zaquestion/lab) [![codecov](https://codecov.io/gh/zaquestion/lab/branch/master/graph/badge.svg)](https://codecov.io/gh/zaquestion/lab) [![Join the chat at https://gitter.im/labcli](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/labcli)

lab wraps git or [hub](https://github.com/github/hub) and adds additional features to make working with GitLab smoother

```
$ lab clone gitlab-com/infrastructure

# expands to:
$ git clone git@gitlab.com:gitlab-com/infrastructure
```

## hub + lab = hublab??

lab will look for hub and uses that as your git binary when available so you don't have to give up hub to use lab
```
$ lab version
git version 2.11.0
hub version 2.3.0-pre9
lab version 0.7.0
```

# Inspiration

The [hub](https://github.com/github/hub) tool made my life significantly easier and still does! lab is heavily inspired by hub and attempts to provide a similar feel.

# Installation

Dependencies

* `git` or `hub`

### Homebrew
```
brew install zaquestion/tap/lab
```

### Source

Required
* [Go 1.9+](https://golang.org/doc/install)
```
$ bash <(curl -s https://raw.githubusercontent.com/zaquestion/lab/master/install.sh)
```

# Configuration

The first time you run lab it will prompt for your GitLab information. lab uses HCL for its config and looks in `~/.config/lab.hcl` and `./lab.hcl`
```
$ lab
Enter default GitLab host (default: https://gitlab.com):
Enter default GitLab user: zaq
Enter default GitLab token:
```

# Aliasing

Like hub, lab feels best when aliased as `git`. In your `.bashrc` or `.bash_profile`
```
alias git=lab
```
