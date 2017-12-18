#/usr/local/env sh

set -e
[[ -z $DEBUG ]] || set -x



(go get -u -d github.com/zaquestion/lab &&
cd $GOPATH/src/github.com/zaquestion/lab &&
go get -u github.com/golang/dep/cmd/dep &&
dep ensure &&
go install -ldflags "-X \"main.version=$(git  rev-parse --short=10 HEAD)\""  github.com/zaquestion/lab)
