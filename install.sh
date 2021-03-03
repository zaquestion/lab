#!/bin/sh
# Usage: [sudo] [BINDIR=/usr/local/bin] ./install.sh [<BINDIR>]
#
# Example:
#     1. sudo ./install.sh /usr/local/bin
#     2. sudo ./install.sh /usr/bin
#     3. ./install.sh $HOME/usr/bin
#     4. BINDIR=$HOME/usr/bin ./install.sh
#
# Default BINDIR=/usr/bin

set -euf

if [ -n "${DEBUG-}" ]; then
    set -x
fi

: "${BINDIR:=/usr/bin}"

if [ $# -gt 0 ]; then
  BINDIR=$1
fi

_can_install() {
  if [ ! -d "${BINDIR}" ]; then
    mkdir -p "${BINDIR}" 2> /dev/null
  fi
  [ -d "${BINDIR}" ] && [ -w "${BINDIR}" ]
}

if ! _can_install && [ "$(id -u)" != 0 ]; then
  printf "Run script as sudo\n"
  exit 1
fi

if ! _can_install; then
  printf -- "Can't install to %s\n" "${BINDIR}"
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
        printf "OS not supported\n"
        exit 1
        ;;
esac

printf "Fetching latest version\n"
latest="$(curl -sL 'https://api.github.com/repos/zaquestion/lab/releases/latest' | grep 'tag_name' | grep -o 'v[0-9\.]\+' | cut -c 2-)"
tempFolder="/tmp/lab_v${latest}"

printf -- "Found version %s\n" "${latest}"

mkdir -p "${tempFolder}" 2> /dev/null
printf -- "Downloading lab_%s_%s_%s.tar.gz\n" "${latest}" "${os}" "${machine}"
curl -sL "https://github.com/zaquestion/lab/releases/download/v${latest}/lab_${latest}_${os}_${machine}.tar.gz" | tar -C "${tempFolder}" -xzf -

printf -- "Installing...\n"
install -m755 "${tempFolder}/lab" "${BINDIR}/lab"

printf "Cleaning up temp files\n"
rm -rf "${tempFolder}"

printf -- "Successfully installed lab in %s/\n" "${BINDIR}"
