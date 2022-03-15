.PHONY: checks
checks: dependencies
	@test -z $(shell gofmt -l -s $(shell go list -f '{{.Dir}}' ./...) | tee /dev/stderr) || (echo "Fix formatting issues"; exit 1)
	find . -name '*.go' | xargs addlicense -check || (echo "Missing license headers"; exit 1)
	@go vet -all $(shell go list -f '{{.Dir}}' ./...)
	@ineffassign $(shell go list -f '{{.Dir}}' ./...)

.PHONY: lint
lint:
	@golint $(shell go list -f '{{.Dir}}' ./...)

.PHONY: gocyclo
gocyclo:
	@gocyclo -over 15 $(shell go list -f '{{.Dir}}' ./...)

.PHONY: ineffassign
ineffassign:
	@ineffassign $(shell go list -f '{{.Dir}}' ./...)

.PHONY: misspell
misspell:
	@misspell $(shell go list -f '{{.Dir}}' ./...)

.PHONY: unit-tests
unit-tests:
	@go test -cover $(shell go list ./... | grep -v '/integration/')
	cd integration/nwo/; go test -cover ./...

.PHONY: unit-tests-race
unit-tests-race:
	@export GORACE=history_size=7; go test -race -cover $(shell go list ./... | grep -v '/integration/')
	cd integration/nwo/; go test -cover ./...

.PHONY: docker-images
docker-images:
	docker pull hyperledger/fabric-baseos:2.2
	docker image tag hyperledger/fabric-baseos:2.2 hyperledger/fabric-baseos:latest
	docker pull hyperledger/fabric-ccenv:2.2
	docker image tag hyperledger/fabric-ccenv:2.2 hyperledger/fabric-ccenv:latest
	docker pull couchdb:3.1.1
	docker pull confluentinc/cp-kafka:5.3.1
	docker pull confluentinc/cp-zookeeper:5.3.1

.PHONY: dependencies
dependencies:
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/gordonklaus/ineffassign
	go get -u github.com/google/addlicense

.PHONY: integration-tests-tcc-dlog
integration-tests-tcc-dlog: docker-images dependencies
	cd ./integration/token/tcc/basic/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-fabtoken
integration-tests-tcc-fabtoken: docker-images dependencies
	cd ./integration/token/tcc/basic/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-dvp-fabtoken
integration-tests-tcc-dvp-fabtoken: docker-images dependencies
	cd ./integration/token/tcc/dvp/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-dvp-dlog
integration-tests-tcc-dvp-dlog: docker-images dependencies
	cd ./integration/token/tcc/dvp/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: clean
clean:
	docker network prune -f
	docker container prune -f
	rm -rf ./integration/token/tcc/basic/dlog/cmd/
	rm -rf ./integration/token/tcc/basic/fabtoken/cmd/
	rm -rf ./integration/token/tcc/dvp/dlog/cmd/
	rm -rf ./integration/token/tcc/dvp/fabtoken/cmd/
	rm -rf ./samples/fabric/fungible/cmd
	rm -rf ./samples/fabric/dvp/cmd

.PHONY: tokengen
tokengen:
	@go install ./cmd/tokengen
