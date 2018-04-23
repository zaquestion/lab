install:
	dep ensure
	go install -ldflags "-X \"main.version=$$(git  rev-parse --short=10 HEAD)\""  github.com/zaquestion/lab

test:
	bash -c "trap 'trap - SIGINT SIGTERM ERR; mv testdata/.git testdata/test.git; rm coverage-* 2>&1 > /dev/null; exit 1' SIGINT SIGTERM ERR; $(MAKE) internal-test"

internal-test:
	rm coverage-* 2>&1 > /dev/null || true
	mv testdata/test.git testdata/.git
	go test -coverprofile=coverage-main.out -covermode=count -coverpkg ./... -run=$(run) github.com/zaquestion/lab/cmd github.com/zaquestion/lab/internal/...
	mv testdata/.git testdata/test.git
	go get github.com/wadey/gocovmerge
	gocovmerge coverage-*.out > coverage.txt && rm coverage-*.out

.PHONY: install test internal-test
