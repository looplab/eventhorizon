default: run

.PHONY: run
run: build_frontend
	docker run -d --name mongo -p 27017:27017 mongo:4.2
	go run backend/main.go

.PHONY: build_frontend
build_frontend:
	cd frontend && elm make src/Main.elm --output=elm.js

.PHONY: stop
stop:
	docker stop mongo
	docker rm mongo
