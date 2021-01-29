default: test

.PHONY: test
test:
	go test -race -short ./...

.PHONY: test_docker
test_docker:
	docker-compose run --rm golang make test

.PHONY: test_cover
test_cover:
	go list -f '{{if len .TestGoFiles}}"go test -race -short -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c

.PHONY: test_integration
test_integration:
	go test -race -run Integration ./...

.PHONY: test_integration_docker
test_integration_docker: run
	docker-compose run --rm golang make test_integration

.PHONY: test_integration_cover
test_integration_cover:
	go list -f '{{if len .TestGoFiles}}"go test -race -run Integration -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c

.PHONY: test_loadtest
test_loadtest:
	go test -race -v -run Loadtest ./...

.PHONY: test_loadtest_docker
test_loadtest_docker: run
	docker-compose run --rm golang make test_loadtest

.PHONY: publish_cover
publish_cover:
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls
	gover
	goveralls -coverprofile=gover.coverprofile -service=travis-ci -repotoken="$$COVERALLS_TOKEN"

.PHONY: run
run:
	docker-compose up -d mongo gpubsub kafka
	sleep 15

.PHONY: run_mongodb
run_mongodb:
	docker-compose up -d mongo

.PHONY: run_gpubsub
run_gpubsub:
	docker-compose up -d gpubsub

.PHONY: run_kafka
run_kafka:
	docker-compose up -d kafka

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
