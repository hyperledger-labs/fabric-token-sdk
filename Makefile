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

.PHONY: integration-tests-dlog-fabric
integration-tests-dlog-fabric: docker-images dependencies
	cd ./integration/token/fungible/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-fabtoken-fabric
integration-tests-fabtoken-fabric: docker-images dependencies
	cd ./integration/token/fungible/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-dlog-orion
integration-tests-dlog-orion: docker-images orion-server-images dependencies
	cd ./integration/token/fungible/odlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-fabtoken-orion
integration-tests-fabtoken-orion: docker-images orion-server-images dependencies
	cd ./integration/token/fungible/ofabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-nft-dlog
integration-tests-nft-dlog: docker-images dependencies
	cd ./integration/token/nft/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-nft-fabtoken
integration-tests-nft-fabtoken: docker-images dependencies
	cd ./integration/token/nft/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-nft-dlog-orion
integration-tests-nft-dlog-orion: docker-images orion-server-images dependencies
	cd ./integration/token/nft/odlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-nft-fabtoken-orion
integration-tests-nft-fabtoken-orion: docker-images orion-server-images dependencies
	cd ./integration/token/nft/ofabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-dvp-fabtoken
integration-tests-dvp-fabtoken: docker-images dependencies
	cd ./integration/token/dvp/fabtoken; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: integration-tests-dvp-dlog
integration-tests-dvp-dlog: docker-images dependencies
	cd ./integration/token/dvp/dlog; ginkgo -keepGoing --slowSpecThreshold 60 .

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: clean
clean:
	docker network prune -f
	docker container prune -f
	rm -rf ./integration/token/fungible/dlog/cmd/
	rm -rf ./integration/token/fungible/fabtoken/cmd/
	rm -rf ./integration/token/fungible/odlog/cmd/
	rm -rf ./integration/token/fungible/ofabtoken/cmd/
	rm -rf ./integration/token/nft/dlog/cmd/
	rm -rf ./integration/token/nft/fabtoken/cmd/
	rm -rf ./integration/token/nft/odlog/cmd/
	rm -rf ./integration/token/nft/ofabtoken/cmd/
	rm -rf ./integration/token/dvp/dlog/cmd/
	rm -rf ./integration/token/dvp/fabtoken/cmd/
	rm -rf ./samples/fungible/cmd
	rm -rf ./samples/nft/cmd

.PHONY: tokengen
tokengen:
	@go install ./cmd/tokengen
