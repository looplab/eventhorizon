default: run_services test

.PHONY: test
test:
	go test -race ./...

.PHONY: test_docker
test_docker:
	docker-compose run --rm golang make test

.PHONY: cover
cover:
	go list -f '{{if len .TestGoFiles}}"go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c

.PHONY: publish_cover
publish_cover: cover
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls
	gover
	@goveralls -coverprofile=gover.coverprofile -service=travis-ci -repotoken=$(COVERALLS_TOKEN)

.PHONY: run
run:
	docker-compose up -d

.PHONY: run_mongodb
run_mongodb:
	docker-compose up -d mongo

.PHONY: run_gpubsub
run_gpubsub:
	docker-compose up -d gpubsub

.PHONY: run_kafka
run_kafka:
	docker-compose up -d zookeeper kafka

.PHONY: stop
stop:
	docker-compose down

.PHONY: clean
clean:
	@find . -name \.coverprofile -type f -delete
	@rm -f gover.coverprofile

.PHONY: mongo_shell
mongo_shell:
	docker run -it --network eventhorizon_default --rm mongo:4.2 mongo --host mongo test
