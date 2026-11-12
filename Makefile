include standard.mk

CHECK_EXTRA := modverify build workflows fuzz

.PHONY: modverify build workflows fuzz
modverify:
	$(GO) mod verify

build:
	$(GO) build ./...

workflows:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12

fuzz:
	$(GO) test ./dns -run '^$$' -fuzz '^FuzzParseZoneName$$' -fuzztime=5s -parallel=2
	$(GO) test . -run '^$$' -fuzz '^FuzzValue_UnmarshalJSON$$' -fuzztime=5s -parallel=2
