.PHONY: checks
checks: licensecheck gofmt goimports govet misspell ineffassign staticcheck

.PHONY: licensecheck
licensecheck:
	@echo Running license check
	@find . -name '*.go' | xargs addlicense -check || (echo "Missing license headers"; exit 1)

.PHONY: gofmt
gofmt:
	@echo Running gofmt
	@{ \
	OUTPUT="$$(gofmt -l -s . || true)"; \
	if [ -n "$$OUTPUT" ]; then \
		echo "The following gofmt issues were flagged:"; \
		echo "$$OUTPUT"; \
		echo "The gofmt command 'gofmt -l -s -w' must be run for these files"; \
		exit 1; \
	fi \
	}

.PHONY: goimports
goimports:
	@echo Running goimports
	@{ \
	OUTPUT="$$(goimports -l .)"; \
	if [ -n "$$OUTPUT" ]; then \
    	echo "The following files contain goimports errors"; \
    	echo "$$OUTPUT"; \
    	echo "The goimports command 'goimports -l -w' must be run for these files"; \
    	exit 1; \
	fi \
	}

.PHONY: govet
govet:
	@echo Running go vet
	@go vet -all $(shell go list -f '{{.Dir}}' ./...) || (echo "Found some issues identified by 'go vet -all'. Please fix them!"; exit 1;)

.PHONY: misspell
misspell:
	@echo Running misspell
	@{ \
	OUTPUT="$$(misspell . || true)"; \
	if [ -n "$$OUTPUT" ]; then \
		echo "The following files are have spelling errors:"; \
		echo "$$OUTPUT"; \
		exit 1; \
	fi \
	}

.PHONY: staticcheck
staticcheck:
	@echo Running staticcheck
	@{ \
	OUTPUT="$$(staticcheck -tests=false ./... | grep -v .pb.go || true)"; \
	if [ -n "$$OUTPUT" ]; then \
		echo "The following staticcheck issues were flagged:"; \
		echo "$$OUTPUT"; \
		exit 1; \
	fi \
	}

.PHONY: gocyclo
gocyclo:
	@echo Running gocyclo
	@gocyclo -over 15 $(shell go list -f '{{.Dir}}' ./...) || (echo "Found some code with a Cyclomatic complexity over 15! Better refactor"; exit 1;)

.PHONY: ineffassign
ineffassign:
	@echo Running ineffassign
	@ineffassign $(shell go list -f '{{.Dir}}' ./...)




