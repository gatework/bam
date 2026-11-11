include standard.mk

CHECK_EXTRA := modverify build workflows

.PHONY: modverify build workflows
modverify:
	$(GO) mod verify

build:
	$(GO) build ./...

workflows:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
