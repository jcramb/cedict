# Copyright 2020 John Cramb. All rights reserved.
# Copyright 2024 Sebastian Scheibe. All rights reserved.
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

all: install cedict

install: 
	$(GO) install $(LDFLAGS) ./...

build: 
	$(GO) build $(LDFLAGS) ./...

.PHONY: $(CMD)
$(CMD):
	$(GO) build $(LDFLAGS) ./cmd/$@

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: $(GOLINT)
	$(GOLINT) .

test:
	$(GO) test -race -v -covermode atomic -coverprofile=profile.cov ./...

clean:
	rm -f $(CMD)
	rm -rf ./testdata

$(GOLINT):
	go get -u golang.org/x/lint/golint
