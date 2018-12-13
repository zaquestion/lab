#!/usr/bin/env bash

set -e

if [[ ! -z $DEBUG ]]; then
    set -x
fi

: ${PREFIX:=/usr/local}
BINDIR="$PREFIX/bin"

_can_install() {
  if [[ ! -d "$BINDIR" ]]; then
    mkdir -p "$BINDIR" 2> /dev/null
  fi
  [[ -d "$BINDIR" && -w "$BINDIR" ]]
}

if [[ $EUID != 0 ]]; then
    sudo "$0" "$@"
    exit "$?"
fi

if ! _can_install; then
  echo "Can't install to $BINDIR"
  exit 1
fi

case "$(uname -m)" in
    x86_64)
        machine="amd64"
        ;;
    i386)
        machine="386"
        ;;
    *)
        machine=""
        ;;
esac

case $(uname -s) in
    Linux)
        os="linux"
        ;;
    Darwin)
        os="darwin"
        ;;
    *)
        echo "OS not supported"
        exit 1
        ;;
esac

latest="$(curl -sL 'https://api.github.com/repos/zaquestion/lab/releases/latest' | grep 'tag_name' | grep --only 'v[0-9\.]\+' | cut -c 2-)"

curl -sL "https://github.com/zaquestion/lab/releases/download/v${latest}/lab_${latest}_${os}_${machine}.tar.gz" | tar -C /tmp/ -xzf -
cp /tmp/lab $BINDIR/lab
echo "Successfully installed lab into $BINDIR/"
