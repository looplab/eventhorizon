default: build

.PHONY: build
build:
	docker-compose build todomvc

.PHONY: run
run:
	docker-compose up todomvc

.PHONY: run_services
run_services:
	docker-compose up -d mongo gpubsub tracing

.PHONY: stop
stop:
	docker-compose down

.PHONY: run_backend
run_backend:
	go run -v backend/*.go

.PHONY: build_frontend
build_frontend:
	cd frontend && elm make src/Main.elm --output=elm.js
