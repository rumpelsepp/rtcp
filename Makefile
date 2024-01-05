# SPDX-FileCopyrightText: Stefan Tatschner
#
# SPDX-License-Identifier: MIT

GO ?= go

.PHONY: rtcp
rtcp:
	$(GO) build $(GOFLAGS) -o $@ .

