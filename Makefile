test:
	  bash -c "trap 'trap - SIGINT SIGTERM ERR; mv testdata/.git testdata/test.git; exit 1' SIGINT SIGTERM ERR; $(MAKE) internal-test"

internal-test:
	mv testdata/test.git testdata/.git
	go test ./...
	mv testdata/.git testdata/test.git
