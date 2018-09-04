default: test

test: services
	go list -f '{{if len .TestGoFiles}}"go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
.PHONY: test

cover: test
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls
	gover
	@goveralls -coverprofile=gover.coverprofile -service=travis-ci -repotoken=$(COVERALLS_TOKEN)
.PHONY: cover

services:
	docker-compose up -d mongo redis dynamodb gpubsub
.PHONY: services

stop:
	docker-compose down
.PHONY: stop

test_docker: services
	docker-compose run --rm golang make test
.PHONY: test_docker

cover_docker: services
	docker-compose run --rm golang make cover
.PHONY: cover_docker

clean:
	@find . -name \.coverprofile -type f -delete
	@rm -f gover.coverprofile
.PHONY: clean
