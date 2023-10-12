<p align="center">
    <p align="center">
        git + <img src="https://user-images.githubusercontent.com/3167497/34473826-40b4987c-ef2c-11e7-90b9-5ff322c4966f.png" width="100" height="100"> = gitlab
    </p>
    <p align="center">
        <a href="https://travis-ci.org/zaquestion/lab">
            <img src="https://travis-ci.org/zaquestion/lab.svg?branch=master" alt="Build Status">
        </a>
        <a href="https://goreportcard.com/report/github.com/zaquestion/lab">
            <img src="https://goreportcard.com/badge/github.com/zaquestion/lab" alt="Go Report Card">
        </a>
        <a href="https://codecov.io/gh/zaquestion/lab">
            <img src="https://codecov.io/gh/zaquestion/lab/branch/master/graph/badge.svg" alt="codecov">
        </a>
        <a href="https://gitter.im/labcli">
            <img src="https://badges.gitter.im/Join%20Chat.svg" alt="Join the chat">
        </a>
    </p>
    <p align="center">
        <a href="https://liberapay.com/zaquestion/donate">
            <img src="https://liberapay.com/assets/widgets/donate.svg" alt="Donate">
        </a>
    </p>
    <p align="center">
        <img src="https://user-images.githubusercontent.com/1964720/42740177-6478d834-8858-11e8-9667-97f193ecb404.gif" align="center">
    </p>
</p>

_lab_ interacts with repositories on GitLab, including creating/editing merge requests, issues, milestones, snippets
and CI pipelines.

The development team has focused on keeping _lab_ a simple and intuitive command line interface for commands provided
in the [GitLab API](https://docs.gitlab.com/ee/api/api_resources.html). _lab_'s aim is to provide GitLab users an
experience similar to the GitLab WebUI with respect to errors and messages.

# Usage recommendation

One can use _lab_ as a Git alias, integrating seamlessly to their Git workflow.

```
$ cat ~/.gitconfig
...
[alias]
    lab = "!lab"
    lab-i = "!lab issue"
    li = "!lab issue"

$ git lab mr list
$ git lab-i close
$ git li create
```

Also, _lab_ can be set as shell aliases:

```bash
alias mrlist="lab mr list"
```

# Installation

In compilation-time, _lab_ depends only on other Go external modules, defined at go.mod. At runtime, _git_ is required
as many git commands are used by _lab_ to retrieve local repository information.

### Homebrew
```
brew install lab
```

### NixOS
```
nix-env -f '<nixpkgs>' -iA gitAndTools.lab
```

### Scoop
```
scoop install lab
```

### Alpine
```
apk add lab
```

### Bash
In case you don't want to install _lab_ using any of the above package managers you can use the Bash script available:

> :warning: The script will install _lab_ into `/usr/local/bin/`.

```
curl -s https://raw.githubusercontent.com/zaquestion/lab/master/install.sh | sudo bash
```

> :warning: Please take care when executing scripts in this fashion. Make sure you trust the developer providing the
> script and consider peeking at the install script itself (ours is pretty simple ;)

### PreBuilt Binaries

Head to the [releases](https://github.com/zaquestion/lab/releases) page and download your preferred release.

### Source

For building _lab_ from source it's required [Go 1.17+](https://golang.org/doc/install). Older versions (ie. 1.15)
might still be able to build _lab_, but warnings related to unsupported `go.mod` format might be prompted.

```
git clone git@github.com:zaquestion/lab
cd lab
go install -ldflags "-X \"main.version=$(git rev-parse --short=10 HEAD)\"" .
```

or

```
make install
```

# Configuration

_lab_ needs your GitLab information in order to interact with to your GitLab instance. There are several ways to
provide this information to `lab`:

### Fresh start

When _lab_ is executed for the first time, no suitable configuration found, it will prompt for your GitLab information.
For example:

```
$ lab
Enter default GitLab host (default: https://gitlab.com):
Enter default GitLab token:
```

These informations are going to be save it into `~/.config/lab/lab.toml` and won't be asked again.
In case multiple projects require different information (ie. _gitlab.com_ and a self-hosted GitLab service), using
different configuration files as explained in the section below.

### Configuration file

The most common option is to use _lab_ configuration files, which can be placed in different places in an hierarchical
style. The [Tom's Obvious, Minimal Language (TOML)](https://github.com/toml-lang/toml) is used for the configuration
file.

When a local configuration file is present (`./lab.toml`), no other configuration file will be checked, this will be
the only one used for looking up required information.

However, other two options are possible:

1. User-specific: `~/.config/lab/lab.toml`
2. Worktree-specific: `.git/lab/lab.toml`

These two files are merged before _lab_ runs, meaning that they're complementary to each other.  One thing is important
to note though, options set in the worktree configuration file overrides (take precedence over) any value set in the
user-specific file.

An example of user-specific configurations can be found below. As seen in the example file below, any command options
specified by `--help` (for example `lab mr show --help`, or `lab issue edit --help`) can be set in the configuration
file using TOML format.

```toml
[core]
  host = "https://gitlab.com"
  token = "1223334444555556789K"
  user = "yourusername"

[mr_checkout]
  force = true

[mr_create]
  force-linebreak = true
  draft = true

[mr_edit]
  force-linebreak = true
```

### Local environment variables

If running _lab_ locally, the variables `LAB_CORE_HOST` and `LAB_CORE_TOKEN` can be used, preventing configuration file
creation/update. For example to use _gitlab.com_ do:

```
export LAB_CORE_HOST="https://gitlab.com"
```

### CI environment variables

The environment variables `CI_PROJECT_URL`, `CI_JOB_TOKEN` and `GITLAB_USER_LOGIN`, intended to be used in a CI
environment, can be set to prevent any configuration file creation/update. Also, any of these take precedence over all
other options.

# Completions

_lab_ provides completions for [Bash], [Elvish], [Fish], [Oil], [Powershell], [Xonsh] and [Zsh].
Scripts can be directly sourced (though using pre-generated versions is recommended to avoid shell startup delay):

```sh
# bash (~/.bashrc)
source <(lab completion)

# elvish (~/.elvish/rc.elv)
eval (lab completion|slurp)

# fish (~/.config/fish/config.fish)
lab completion | source

# oil
source <(lab completion)

# powershell (~/.config/powershell/Microsoft.PowerShell_profile.ps1)
Set-PSReadlineKeyHandler -Key Tab -Function MenuComplete
lab completion | Out-String | Invoke-Expression

# xonsh (~/.config/xonsh/rc.xsh)
COMPLETIONS_CONFIRM=True
exec($(lab completion xonsh))

# zsh (~/.zshrc)
source <(lab completion zsh)
```

# Contributing

We welcome all contributors and their contributions to _lab_! See the [contribution guide](CONTRIBUTING.md).

# What about [GLab](https://github.com/profclems/glab)?

Both [glab] and `lab` are open-source tools with the same goal of bringing GitLab to your command line and simplifying
the developer workflow. In many ways `lab` is to [hub], what [glab] is to [gh].

If you're looking for a tool like _hub_, less opinionated, that feels like using _git_ and allows you to interact with
GitLab then _lab_ is for you. However, if you're looking for a more opinionated tool intended to simplify your GitLab
workflows, you might consider using [glab].

<p align="center"><img src="https://user-images.githubusercontent.com/2358914/34196973-420d389a-e519-11e7-92e6-3a1486d6b280.png" align="center"></p>

<p xmlns:dct="http://purl.org/dc/terms/">
  <a rel="license"
     href="http://creativecommons.org/publicdomain/zero/1.0/">
    <img src="https://licensebuttons.net/p/zero/1.0/88x31.png" style="border-style: none;" alt="CC0" />
  </a>
  <br />
  To the extent possible under law,
  <a rel="dct:publisher"
     href="https://github.com/zaquestion/lab">
    <span property="dct:title">Zaq? Wiedmann</span></a>
  has waived all copyright and related or neighboring rights to
  <span property="dct:title">Lab</span>.
  This work is published from:
<span property="vcard:Country" datatype="dct:ISO3166"
      content="US" about="https://github.com/zaquestion/lab">
  United States</span>.
</p>




[Bash]:https://www.gnu.org/software/bash/
[Elvish]:https://elv.sh/
[Fish]:https://fishshell.com/
[Oil]:http://www.oilshell.org/
[Powershell]:https://microsoft.com/powershell
[Xonsh]:https://xon.sh/
[Zsh]:https://www.zsh.org/

[gh]:https://github.com/cli/cli
[hub]:https://github.com/github/hub
[lab]:https://github.com/zaquestion/lab
[glab]:https://github.com/profclems/glab
