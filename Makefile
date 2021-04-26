default: test

.PHONY: test
test:
	go test -v -race -short ./...

.PHONY: test_cover
test_cover:
	go list -f '{{if len .TestGoFiles}}"cd {{.Dir}} && go test -v -race -short -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
	go run ./hack/coverage/coverage.go . unit.coverprofile
	@find . -name \.coverprofile -type f -delete

.PHONY: test_integration
test_integration:
	go test -v -race -run Integration ./...

.PHONY: test_integration_cover
test_integration_cover:
	go list -f '{{if len .TestGoFiles}}"cd {{.Dir}} && go test -v -race -run Integration -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
	go run ./hack/coverage/coverage.go . integration.coverprofile
	@find . -name \.coverprofile -type f -delete

.PHONY: test_loadtest
test_loadtest:
	go test -race -v -run Loadtest ./...

.PHONY: test_all_docker
test_all_docker:
	docker-compose up --build --force-recreate eventhorizon-test

.PHONY: merge_coverage
merge_coverage:
	go run ./hack/coverage/coverage.go .

.PHONY: upload_coverage
upload_coverage:
	go run github.com/mattn/goveralls -coverprofile=coverage.out -repotoken="$$COVERALLS_TOKEN"

.PHONY: run
run:
	docker-compose up -d mongodb gpubsub kafka redis jetstream

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

.PHONY: stop
stop:
	docker-compose down

.PHONY: clean
clean:
	@find . -name \.coverprofile -type f -delete
	@rm -f coverage.out

.PHONY: mongodb_shell
mongodb_shell:
	docker run -it --network eventhorizon_default --rm mongo:4.4 mongo --host mongodb-rs test
