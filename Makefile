GO ?= go

rtcp:
	$(GO) build $(GOFLAGS) -o $@ .

.PHONY: rtcp
