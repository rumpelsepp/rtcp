# SPDX-FileCopyrightText: Stefan Tatschner
#
# SPDX-License-Identifier: MIT

GO ?= go

rtcp:
	$(GO) build $(GOFLAGS) -o $@ .

.PHONY: rtcp
