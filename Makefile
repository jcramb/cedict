# Copyright 2020 John Cramb. All rights reserved.
# Licensed under the MIT License. See LICENSE in the project root
# for license information.

# tools
GO = go
GOMOD = $(GO) mod
GOLINT = golint
GOLIST = $(shell $(GO) list ./... | grep -v vendor)

# bins
CMD = cedict

# src
REPO  = github.com/Ecostack/cedict
LDFLAGS= -ldflags="-s -w"

.PHONY: all
all: install cedict

.PHONY: install
install: 
	$(GO) install $(LDFLAGS) ./...

.PHONY: build
build: 
	$(GO) build $(LDFLAGS) ./...

.PHONY: $(CMD)
$(CMD):
	$(GO) build $(LDFLAGS) ./cmd/$@

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint: $(GOLINT)
	$(GOLINT) .

.PHONY: test
test:
	$(GO) test -race -v -covermode atomic -coverprofile=profile.cov ./...

.PHONY: clean
clean:
	rm -f $(CMD)
	rm -rf ./testdata

$(GOLINT):
	go get -u golang.org/x/lint/golint
