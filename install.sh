#!/usr/bin/env bash

set -e

if [[ ! -z $DEBUG ]]; then
    set -x
fi

if [[ $EUID != 0 ]]; then
    sudo "$0" "$@"
    exit "$?"
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
cp /tmp/lab /usr/local/bin/lab
echo "Successfully installed lab into /usr/local/bin/"
