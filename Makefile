PERM_ID ?= `id -u`

MKFILE_DIR = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
COMPOSE_RUN_GOLANG = docker-compose run --rm golang
COMPOSE_UP = docker-compose up -d
COMPOSE_DOWN = docker-compose down
COMPOSE_PULL = docker-compose pull

deps: dotenv
	$(COMPOSE_RUN_GOLANG) make _deps
.PHONY: deps

test: dotenv
	$(COMPOSE_RUN_GOLANG) make _test
.PHONY: test

cover: dotenv
	$(COMPOSE_RUN_GOLANG) make _cover
.PHONY: cover

dev: dotenv deps
	$(COMPOSE_PULL)
	$(COMPOSE_UP)
.PHONY: dev

clean: dotenv
	$(COMPOSE_DOWN)
	@rm -rf .env
	@find . -name \.coverprofile -type f -delete
	@rm -f gover.coverprofile
.PHONY: clean

# replaces .env with DOTENV if the variable is specified
dotenv:
ifdef DOTENV
	cp -f $(DOTENV) .env
else
	$(MAKE) .env
endif

# creates .env from .env.template if it doesn't exist already
.env:
	cp -f .env.template .env
	@echo "PERM_ID=$(PERM_ID)" >> .env

# Helpers
shellGo: dotenv
	$(COMPOSE_RUN_GOLANG) /bin/bash
.PHONY: shellGo

####################################################
# Internal targets
####################################################

_deps:
	dep ensure -v -vendor-only
	$(call fixPermissions)

_test:
	go list -f '{{if len .TestGoFiles}}"GOCACHE=off go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c

_cover:
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls
	gover
	@goveralls -coverprofile=gover.coverprofile -service=travis-ci -repotoken=$(COVERALLS_TOKEN)

define fixPermissions
	chown -R $(PERM_ID) $(MKFILE_DIR)
	chown -R $(PERM_ID) ${GOPATH}/pkg/dep
endef
