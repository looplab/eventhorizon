default: run

.PHONY: run
run: build_frontend
	go run backend/main.go

.PHONY: build_frontend
build_frontend:
	cd frontend && elm make src/Main.elm --output=elm.js
