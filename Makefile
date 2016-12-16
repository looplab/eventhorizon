.PHONEY: cover clean

test: docker
	go test -v ./...

test_integration: docker
	go test -v -tags integration ./...

test_wercker:
	wercker build

test_cover: clean docker
	go list -f '{{if len .TestGoFiles}}"go test -v -covermode=count -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
	gover

cover:
	go tool cover -html=gover.coverprofile

docker:
	-docker run -d --name mongo -p 27017:27017 mongo:latest
	-docker run -d --name redis -p 6379:6379 redis:latest
	-docker run -d --name dynamodb -p 8000:8000 peopleperhour/dynamodb:latest
	-docker run -d --name gpubsub -p 8793:8793 google/cloud-sdk:latest gcloud beta emulators pubsub start --host-port=0.0.0.0:8793
	export PUBSUB_EMULATOR_HOST=localhost:8793

clean:
	-find . -name \.coverprofile -type f -delete
	-rm gover.coverprofile
