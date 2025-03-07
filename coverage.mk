.PHONY: compute-coverage
check-coverage:
	go test $(shell go list ./... | grep -v '/integration/') -coverprofile=./cover.out -covermode=atomic -coverpkg=./...

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage
check-coverage: install-go-test-coverage
	go-test-coverage --config=./testcoverage.yml