.PHONY: integration-tests-dlog-fabric
integration-tests-dlog-fabric:
	cd ./integration/token/fungible/dlog; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dlog-fabric-t1
integration-tests-dlog-fabric-t1:
	cd ./integration/token/fungible/dlog; ginkgo $(GINKGO_TEST_OPTS) --focus "Fungible" .

.PHONY: integration-tests-dlog-fabric-t2
integration-tests-dlog-fabric-t2:
	cd ./integration/token/fungible/dlog; ginkgo $(GINKGO_TEST_OPTS) --focus "Fungible with Auditor = Issuer" .

.PHONY: integration-tests-dloghsm-fabric
integration-tests-dloghsm-fabric: install-softhsm
	@echo "Setup SoftHSM"
	@./ci/scripts/setup_softhsm.sh
	@echo "Start Integration Test"
	cd ./integration/token/fungible/dloghsm; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dloghsm-fabric-t1
integration-tests-dloghsm-fabric-t1: install-softhsm
	@echo "Setup SoftHSM"
	@./ci/scripts/setup_softhsm.sh
	@echo "Start Integration Test"
	cd ./integration/token/fungible/dloghsm; ginkgo $(GINKGO_TEST_OPTS) --focus "Fungible with HSM" .

.PHONY: integration-tests-dloghsm-fabric-t2
integration-tests-dloghsm-fabric-t2: install-softhsm
	@echo "Setup SoftHSM"
	@./ci/scripts/setup_softhsm.sh
	@echo "Start Integration Test"
	cd ./integration/token/fungible/dloghsm; ginkgo $(GINKGO_TEST_OPTS) --focus "Fungible with Auditor = Issuer with HSM" .

.PHONY: integration-tests-fabtoken-fabric
integration-tests-fabtoken-fabric:
	cd ./integration/token/fungible/fabtoken; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-dlog-orion
integration-tests-dlog-orion:
	cd ./integration/token/fungible/odlog; ginkgo $(GINKGO_TEST_OPTS) .

.PHONY: integration-tests-fabtoken-orion
integration-tests-fabtoken-orion:
	cd ./integration/token/fungible/ofabtoken; ginkgo $(GINKGO_TEST_OPTS) .
