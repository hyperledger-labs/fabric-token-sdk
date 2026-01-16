.PHONY: fabricx-docker-images
fabricx-docker-images: ## Pull fabric-x images
	docker pull hyperledger/fabric-x-committer-test-node:$(FABRIC_X_COMMITTER_VERSION)

# Make sure you run install-fabricx-tools after `download-fabric` as it overrides configtxgen
.PHONY: install-fabricx-tools
install-fabricx-tools:
	@cd tools; cat fabx_bins_tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % env GOBIN=$(FABX_BINS) go install %

.PHONY: integration-tests-fabricx-dlog-t1
integration-tests-fabricx-dlog-t1:
	make integration-tests-fabricx-dlog TEST_FILTER="T1"

.PHONY: integration-tests-fabricx-dlog
integration-tests-fabricx-dlog:
	cd ./integration/token/fungible/dlogx; export FAB_BINS=$(FABX_BINS); ginkgo $(GINKGO_TEST_OPTS) --label-filter="$(TEST_FILTER)" .
