default: test

.PHONY: test
test:
	go test -race -short -test.paniconexit0 ./...

.PHONY: test_docker
test_docker:
	docker-compose run --rm golang make test

.PHONY: test_cover
test_cover:
	go list -f '{{if len .TestGoFiles}}"go test -race -short -test.paniconexit0 -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c

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

.PHONY: upload_cover
upload_cover:
	go get -d golang.org/x/tools/cmd/cover
	go get github.com/sozorogami/gover
	go get github.com/mattn/goveralls
	gover
	goveralls -coverprofile=gover.coverprofile -repotoken="$$COVERALLS_TOKEN"

.PHONY: run
run:
	docker-compose up -d mongodb gpubsub kafka redis jetstream nats stan

.PHONY: run_mongodb
run_mongodb:
	docker-compose up -d mongodb

.PHONY: run_gpubsub
run_gpubsub:
	docker-compose up -d gpubsub

.PHONY: run_kafka
run_kafka:
	docker-compose up -d kafka

.PHONY: run_redis
run_redis:
	docker-compose up -d redis

.PHONY: run_jetstream
run_jetstream:
	docker-compose up -d jetstream

.PHONY: run_nats
run_nats:
	docker-compose up -d nats

.PHONY: run_stan
run_stan:
	docker-compose up -d stan

.PHONY: stop
stop:
	docker-compose down

.PHONY: clean
clean:
	@find . -name \.coverprofile -type f -delete
	@rm -f gover.coverprofile

.PHONY: mongodb_shell
mongodb_shell:
	docker run -it --network eventhorizon_default --rm mongo:4.4 mongo --host mongodb test
