# pinned versions
FABRIC_VERSION ?= 3.1.1
FABRIC_CA_VERSION ?= 1.5.7
FABRIC_TWO_DIGIT_VERSION = $(shell echo $(FABRIC_VERSION) | cut -d '.' -f 1,2)

# need to install fabric binaries outside of fts tree for now (due to chaincode packaging issues)
FABRIC_BINARY_BASE=$(PWD)/../fabric
FAB_BINS ?= $(FABRIC_BINARY_BASE)/bin

# integration test options
GINKGO_TEST_OPTS ?=
GINKGO_TEST_OPTS += --keep-going

TOP = .

all: install-tools install-softhsm checks unit-tests #integration-tests

.PHONY: install-tools
install-tools:
# Thanks for great inspiration https://marcofranssen.nl/manage-go-tools-via-go-modules
	@echo Installing tools from tools/tools.go
	@cd tools; cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %
	@$(MAKE) install-linter-tool

.PHONY: download-fabric
download-fabric:
	./ci/scripts/download_fabric.sh $(FABRIC_BINARY_BASE) $(FABRIC_VERSION) $(FABRIC_CA_VERSION)

# include the checks target
include $(TOP)/checks.mk
# include the interop target
include $(TOP)/interop.mk
# include the fungible target
include $(TOP)/fungible.mk

.PHONY: unit-tests
unit-tests:
	@go test -cover $(shell go list ./... | grep -v '/integration/')
	cd integration/nwo/; go test -cover ./...
	cd token/services/identity/storage/kvs/hashicorp/; go test -cover ./...

.PHONY: unit-tests-race
unit-tests-race:
	@export GORACE=history_size=7; go test -race -cover $(shell go list ./... | grep -v '/integration/')
	cd integration/nwo/; go test -cover ./...

.PHONY: install-softhsm
install-softhsm:
	./ci/scripts/install_softhsm.sh

.PHONY: docker-images
docker-images: fabric-docker-images monitoring-docker-images testing-docker-images

.PHONY: testing-docker-images
testing-docker-images:
	docker pull postgres:16.2-alpine
	docker tag postgres:16.2-alpine fsc.itests/postgres:latest
	docker pull hashicorp/vault

.PHONY: fabric-docker-images
fabric-docker-images:
	docker pull hyperledger/fabric-baseos:$(FABRIC_TWO_DIGIT_VERSION)
	docker image tag hyperledger/fabric-baseos:$(FABRIC_TWO_DIGIT_VERSION) hyperledger/fabric-baseos:latest
	docker pull hyperledger/fabric-ccenv:$(FABRIC_TWO_DIGIT_VERSION)
	docker image tag hyperledger/fabric-ccenv:$(FABRIC_TWO_DIGIT_VERSION) hyperledger/fabric-ccenv:latest

.PHONY: monitoring-docker-images
monitoring-docker-images:
	docker pull ghcr.io/hyperledger-labs/explorer-db:latest
	docker pull ghcr.io/hyperledger-labs/explorer:latest
	docker pull prom/prometheus:latest
	docker pull grafana/grafana:latest
	docker pull jaegertracing/all-in-one:latest
	docker pull otel/opentelemetry-collector:latest

.PHONY: integration-tests-nft-dlog
integration-tests-nft-dlog:
	cd ./integration/token/nft/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-nft-fabtoken
integration-tests-nft-fabtoken:
	cd ./integration/token/nft/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dvp-fabtoken
integration-tests-dvp-fabtoken:
	cd ./integration/token/dvp/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dvp-dlog
integration-tests-dvp-dlog:
	cd ./integration/token/dvp/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .


.PHONY: tidy
tidy:
	@go mod tidy
	cd tools; go mod tidy
	cd token/services/identity/storage/kvs/hashicorp; go mod tidy

.PHONY: clean
clean:
	docker network prune -f
	docker container prune -f
	docker volume prune -f
	rm -rf ./integration/token/fungible/dlog/cmd/
	rm -rf ./integration/token/fungible/dlog/testdata/
	rm -rf ./integration/token/fungible/dloghsm/cmd/
	rm -rf ./integration/token/fungible/dloghsm/testdata/
	rm -rf ./integration/token/fungible/dlogstress/cmd/
	rm -rf ./integration/token/fungible/dlogstress/testdata/
	rm -rf ./integration/token/fungible/fabtoken/cmd/
	rm -rf ./integration/token/fungible/fabtoken/testdata/
	rm -rf ./integration/token/fungible/odlog/cmd/
	rm -rf ./integration/token/fungible/ofabtoken/cmd/
	rm -rf ./integration/token/fungible/mixed/cmd/
	rm -rf ./integration/token/nft/dlog/cmd/
	rm -rf ./integration/token/nft/fabtoken/cmd/
	rm -rf ./integration/token/nft/odlog/cmd/
	rm -rf ./integration/token/nft/ofabtoken/cmd/
	rm -rf ./integration/token/dvp/dlog/cmd/
	rm -rf ./integration/token/dvp/fabtoken/cmd/
	rm -rf ./integration/token/interop/fabtoken/cmd/
	rm -rf ./integration/token/interop/dlog/cmd/
	rm -rf ./integration/token/fungible/update/cmd/
	rm -rf ./integration/token/fungible/update/testdata/

.PHONY: clean-fabric-peer-images
clean-fabric-peer-images:
	docker images -a | grep "_peer.org" | awk '{print $3}' | xargs docker rmi
	docker images -a | grep "_peer_" | awk '{print $3}' | xargs docker rmi

.PHONY: tokengen
tokengen:
	@go install ./cmd/tokengen

.PHONY: idemixgen
txgen:
	@go install github.com/IBM/idemix/tools/idemixgen

.PHONY: clean-all-containers
clean-all-containers:
	@if [ -n "$$(docker ps -aq)" ]; then docker rm -f $$(docker ps -aq); else echo "No containers to remove"; fi

.PHONY: lint
lint:
	@echo "Running Go Linters..."
	golangci-lint run --color=always --timeout=4m

.PHONY: lint-auto-fix
lint-auto-fix:
	@echo "Running Go Linters with auto-fix..."
	golangci-lint run --color=always --timeout=4m --fix

.PHONY: install-linter-tool
install-linter-tool:
	@echo "Installing golangci Linter"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(HOME)/go/bin v2.1.6
