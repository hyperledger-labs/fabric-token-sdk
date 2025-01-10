# pinned versions
FABRIC_VERSION ?= 2.5.0
FABRIC_CA_VERSION ?= 1.5.7
FABRIC_TWO_DIGIT_VERSION = $(shell echo $(FABRIC_VERSION) | cut -d '.' -f 1,2)
ORION_VERSION=v0.2.10

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

.PHONY: unit-tests-race
unit-tests-race:
	@export GORACE=history_size=7; go test -race -cover $(shell go list ./... | grep -v '/integration/')
	cd integration/nwo/; go test -cover ./...

.PHONY: install-softhsm
install-softhsm:
	./ci/scripts/install_softhsm.sh

.PHONY: docker-images
docker-images: fabric-docker-images orion-server-images monitoring-docker-images testing-docker-images

.PHONY: testing-docker-images
testing-docker-images:
	docker pull postgres:16.2-alpine
	docker tag postgres:16.2-alpine postgres:latest

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

.PHONY: orion-server-images
orion-server-images:
	docker pull orionbcdb/orion-server:$(ORION_VERSION)
	docker image tag orionbcdb/orion-server:$(ORION_VERSION) orionbcdb/orion-server:latest


.PHONY: integration-tests-nft-dlog
integration-tests-nft-dlog:
	cd ./integration/token/nft/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-nft-fabtoken
integration-tests-nft-fabtoken:
	cd ./integration/token/nft/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-nft-dlog-orion
integration-tests-nft-dlog-orion:
	cd ./integration/token/nft/odlog; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-nft-fabtoken-orion
integration-tests-nft-fabtoken-orion:
	cd ./integration/token/nft/ofabtoken; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dvp-fabtoken
integration-tests-dvp-fabtoken:
	cd ./integration/token/dvp/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dvp-dlog
integration-tests-dvp-dlog:
	cd ./integration/token/dvp/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dlogstress-t1
integration-tests-dlogstress-t1:
	make integration-tests-dlogstress TEST_FILTER="T1"

.PHONY: integration-tests-dlogstress-t2
integration-tests-dlogstress-t2:
	make integration-tests-dlogstress TEST_FILTER="T2"

.PHONY: integration-tests-dlogstress
integration-tests-dlogstress:
	cd ./integration/token/fungible/dlogstress; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --label-filter="$(TEST_FILTER)" .

.PHONY: tidy
tidy:
	@go mod tidy
	cd tools; go mod tidy

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

.PHONY: clean-fabric-peer-images
clean-fabric-peer-images:
	docker images -a | grep "_peer_" | awk '{print $3}' | xargs docker rmi
	docker images -a | grep "peer.org" | awk '{print $3}' | xargs docker rmi

.PHONY: tokengen
tokengen:
	@go install ./cmd/tokengen

.PHONY: txgen
txgen:
	@go install ./cmd/txgen