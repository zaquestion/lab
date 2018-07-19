VERSION ?= $(shell git describe --long --tags)
GOURL ?= github.com/zaquestion/lab

deps:
	dep ensure

build: deps
	go build -ldflags "-X \"main.version=$(VERSION)\"" $(GOURL)

install: build
	go install $(GOURL)

test:
	bash -c "trap 'trap - SIGINT SIGTERM ERR; mv testdata/.git testdata/test.git; rm coverage-* 2>&1 > /dev/null; exit 1' SIGINT SIGTERM ERR; $(MAKE) internal-test"

internal-test:
	rm coverage-* 2>&1 > /dev/null || true
	mv testdata/test.git testdata/.git
	go test -coverprofile=coverage-main.out -covermode=count -coverpkg ./... -run=$(run) $(GOURL)/cmd $(GOURL)/internal/...
	mv testdata/.git testdata/test.git
	go get github.com/wadey/gocovmerge
	gocovmerge coverage-*.out > coverage.txt && rm coverage-*.out

.PHONY: deps install test internal-test
