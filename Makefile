.PHONEY: cover clean

test: run_services
	PUBSUB_EMULATOR_HOST=localhost:8793 go test $$(go list ./... | grep -v /vendor/)

test_integration: run_services
	go test -tags integration $$(go list ./... | grep -v /vendor/)

test_wercker:
	wercker build

test_cover: clean run_services
	go list -f '{{if len .TestGoFiles}}"PUBSUB_EMULATOR_HOST=localhost:8793 go test -v -covermode=count -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' $$(go list ./... | grep -v /vendor/) | xargs -L 1 sh -c
	gover

cover:
	go tool cover -html=gover.coverprofile

run_services:
	-docker run -d --name mongo -p 27017:27017 mongo:latest
	-docker run -d --name redis -p 6379:6379 redis:latest
	-docker run -d --name dynamodb -p 8000:8000 peopleperhour/dynamodb:latest
	-docker run -d --name gpubsub -p 8793:8793 google/cloud-sdk:latest gcloud beta emulators pubsub start --host-port=0.0.0.0:8793

update_services:
	docker pull mongo:latest
	docker pull redis:latest
	docker pull peopleperhour/dynamodb:latest
	docker pull google/cloud-sdk:latest

clean:
	-find . -name \.coverprofile -type f -delete
	-rm gover.coverprofile
