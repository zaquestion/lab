VERSION ?= $(shell git describe --long --tags)
GOURL ?= github.com/zaquestion/lab

build:
	GO111MODULE=on go build -ldflags "-X 'main.version=$(VERSION)'" $(GOURL)

install: build
	GO111MODULE=on go install -ldflags "-X 'main.version=$(VERSION)'" $(GOURL)

test:
	bash -c "trap 'trap - SIGINT SIGTERM ERR; mv testdata/.git testdata/test.git; rm coverage-* 2>&1 > /dev/null; exit 1' SIGINT SIGTERM ERR; $(MAKE) internal-test"

internal-test:
	rm -f coverage-*
	GO111MODULE=on go test -coverprofile=coverage-main.out -covermode=count -coverpkg ./... -run=$(run) $(GOURL)/cmd $(GOURL)/internal/...
	go install -mod=readonly github.com/wadey/gocovmerge
	$(GOPATH)/bin/gocovmerge coverage-*.out > coverage.txt && rm coverage-*.out

.PHONY: build install test internal-test
