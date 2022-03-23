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

.PHONY: monitoring-docker-images
monitoring-docker-images:
	docker pull hyperledger/explorer-db:latest
	docker pull hyperledger/explorer:latest
	docker pull prom/prometheus:latest
	docker pull grafana/grafana:latest

.PHONY: orion-server-images
orion-server-images:
	docker pull orionbcdb/orion-server:latest

.PHONY: dependencies
dependencies:
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/gordonklaus/ineffassign
	go get -u github.com/google/addlicense

.PHONY: integration-tests-tcc-dlog-fabric
integration-tests-tcc-dlog-fabric: docker-images dependencies
	cd ./integration/token/tcc/fungible/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-fabtoken-fabric
integration-tests-tcc-fabtoken-fabric: docker-images dependencies
	cd ./integration/token/tcc/fungible/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-nft-dlog
integration-tests-tcc-nft-dlog: docker-images dependencies
	cd ./integration/token/tcc/nft/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-nft-fabtoken
integration-tests-tcc-nft-fabtoken: docker-images dependencies
	cd ./integration/token/tcc/nft/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-dlog-orion
integration-tests-tcc-dlog-orion: docker-images orion-server-images dependencies
	cd ./integration/token/tcc/basic/odlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-tcc-fabtoken-orion
integration-tests-tcc-fabtoken-orion: docker-images orion-server-images dependencies
	cd ./integration/token/tcc/basic/ofabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

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
	rm -rf ./integration/token/tcc/fungible/dlog/cmd/
	rm -rf ./integration/token/tcc/fungible/fabtoken/cmd/
	rm -rf ./integration/token/tcc/nft/dlog/cmd/
	rm -rf ./integration/token/tcc/nft/fabtoken/cmd/
	rm -rf ./integration/token/tcc/basic/odlog/cmd/
	rm -rf ./integration/token/tcc/basic/ofabtoken/cmd/
	rm -rf ./integration/token/tcc/dvp/dlog/cmd/
	rm -rf ./integration/token/tcc/dvp/fabtoken/cmd/
	rm -rf ./samples/fabric/fungible/cmd
	rm -rf ./samples/fabric/dvp/cmd

.PHONY: tokengen
tokengen:
	@go install ./cmd/tokengen
