# git + lab = gitlab [![Build Status](https://travis-ci.org/zaquestion/lab.svg?branch=master)](https://travis-ci.org/zaquestion/lab) [![BCH compliance](https://bettercodehub.com/edge/badge/zaquestion/lab?branch=master)](https://bettercodehub.com/)

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
lab version 0.1.0-alpha
```

# Inspiration

The [hub](https://github.com/github/hub) tool made my life significantly easier and still does! lab is heavily inspired by hub and attempts to provide a similar feel.

# Installation

Dependencies

* git or hub

```
$ go get github.com/zaquestion/lab

$ lab version
git version 2.11.0
lab version 0.1.0-alpha
```

The first time you run lab it will prompt for your GitLab information. All configuration is managed through `git config` so don't worry if you mess it up. Keys can be set at the system, global, or local level.
```
$ lab
Enter default GitLab host (default: https://gitlab.com):
Enter default GitLab user: zaq
Enter default GitLab token (default: empty):
```

Relevant lab `git config` keys:
* gitlab.host
* gitlab.user
* gitlab.token


# Aliasing

Like hub, lab feels best when aliased as `git`. However lab is still in early development _some_ clone or fork commands could be broken when falling through to hub/git

In your `.bashrc` or `.bash_profile`
```
alias git=lab
```
