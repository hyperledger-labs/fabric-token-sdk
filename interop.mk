.PHONY: integration-tests-interop-fabtoken
integration-tests-interop-fabtoken:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-interop-dlog
integration-tests-interop-dlog:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-interop-dlog-t1
integration-tests-interop-dlog-t1:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Single Fabric Network" .

.PHONY: integration-tests-interop-dlog-t2
integration-tests-interop-dlog-t2:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Single Orion Network" .

.PHONY: integration-tests-interop-dlog-t3
integration-tests-interop-dlog-t3:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t4
integration-tests-interop-dlog-t4:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "Fast Exchange Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t5
integration-tests-interop-dlog-t5:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC No Cross Claim Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t6
integration-tests-interop-dlog-t6:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC No Cross Claim with Orion and Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t7
integration-tests-interop-dlog-t7:
	cd ./integration/token/interop/dlog; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "Asset Transfer With Two Fabric Networks" .

.PHONY: integration-tests-interop-fabtoken-t1
integration-tests-interop-fabtoken-t1:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Single Fabric Network" .

.PHONY: integration-tests-interop-fabtoken-t2
integration-tests-interop-fabtoken-t2:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Single Orion Network" .

.PHONY: integration-tests-interop-fabtoken-t3
integration-tests-interop-fabtoken-t3:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC Two Fabric Networks" .

.PHONY: integration-tests-interop-fabtoken-t4
integration-tests-interop-fabtoken-t4:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "Fast Exchange Two Fabric Networks" .

.PHONY: integration-tests-interop-fabtoken-t5
integration-tests-interop-fabtoken-t5:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC No Cross Claim Two Fabric Networks" .

.PHONY: integration-tests-interop-fabtoken-t6
integration-tests-interop-fabtoken-t6:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "HTLC No Cross Claim with Orion and Fabric Networks" .

.PHONY: integration-tests-interop-fabtoken-t7
integration-tests-interop-fabtoken-t7:
	cd ./integration/token/interop/fabtoken; export FAB_BINS=$(FAB_BINS); ginkgo $(GINKGO_TEST_OPTS) --focus "Asset Transfer With Two Fabric Networks" .
