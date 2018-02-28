PKGNAME = github.com/eXeC64/nanolist

GOPATH = $(shell pwd)/.go
PKGPATH = .go/src/$(PKGNAME)

all: nanolist

.go:
	mkdir -p $(dir $(PKGPATH))
	ln -fs $(shell dirname $(GOPATH)) $(PKGPATH)

get: .go
	env GOPATH=$(GOPATH) go get -d ./...

test: .go
	env GOPATH=$(GOPATH) go test ./...

nanolist: .go
	env GOPATH=$(GOPATH) go build -o $@ ./cmd/$@

clean:
	rm -rf .go nanolist

.PHONY: nanolist get test clean
