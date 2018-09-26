default: services test

test:
	go test ./...
.PHONY: test

test_docker:
	docker-compose run --rm golang make test
.PHONY: test_docker

cover:
	go list -f '{{if len .TestGoFiles}}"go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
.PHONY: cover

publish_cover: cover
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls
	gover
	@goveralls -coverprofile=gover.coverprofile -service=travis-ci -repotoken=$(COVERALLS_TOKEN)
.PHONY: publish_cover

services:
	docker-compose pull mongo redis gpubsub
	docker-compose up -d mongo redis gpubsub
.PHONY: services

stop:
	docker-compose down
.PHONY: stop

clean:
	@find . -name \.coverprofile -type f -delete
	@rm -f gover.coverprofile
.PHONY: clean
