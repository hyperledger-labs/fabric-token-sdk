.PHONY: integration-tests-interop-fabtoken
integration-tests-interop-fabtoken:
	cd ./integration/token/interop/fabtoken; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-interop-dlog
integration-tests-interop-dlog:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-interop-dlog-t1
integration-tests-interop-dlog-t1:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "HTLC Single Fabric Network" .

.PHONY: integration-tests-interop-dlog-t2
integration-tests-interop-dlog-t2:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "HTLC Single Orion Network" .

.PHONY: integration-tests-interop-dlog-t3
integration-tests-interop-dlog-t3:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "HTLC Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t4
integration-tests-interop-dlog-t4:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "Fast Exchange Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t5
integration-tests-interop-dlog-t5:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "HTLC No Cross Claim Two Fabric Networks" .

.PHONY: integration-tests-interop-dlog-t6
integration-tests-interop-dlog-t6:
	cd ./integration/token/interop/dlog; ginkgo $(GINKGO_TEST_OPTS) -ginkgo.focus "HTLC No Cross Claim with Orion and Fabric Networks" .
