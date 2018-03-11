test:
	  bash -c "trap 'trap - SIGINT SIGTERM ERR; mv testdata/.git testdata/test.git; rm coverage-* 2>&1 > /dev/null; exit 1' SIGINT SIGTERM ERR; $(MAKE) internal-test"

internal-test:
	dep ensure
	rm coverage-* 2>&1 > /dev/null || true
	mv testdata/test.git testdata/.git
	go test -coverprofile=coverage-git.out -covermode=count github.com/zaquestion/lab/internal/git
	go test -coverprofile=coverage-gitlab.out -covermode=count github.com/zaquestion/lab/internal/gitlab
	go test -coverprofile=coverage-cmd.out -covermode=count -coverpkg ./... github.com/zaquestion/lab/cmd
	mv testdata/.git testdata/test.git
	go get github.com/wadey/gocovmerge
	gocovmerge coverage-*.out > coverage.txt && rm coverage-*.out
