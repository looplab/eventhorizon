.PHONEY: cover clean

test: docker
	go test -v ./...

test_cover: docker
	go list -f '{{if len .TestGoFiles}}"go test -v -covermode=count -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' ./... | xargs -L 1 sh -c
	gover

cover:
	go tool cover -html=gover.coverprofile

docker:
	-docker run -d --name mongo -p 27017:27017 mongo
	-docker run -d --name redis -p 6379:6379 redis

clean:
	find . -name \.coverprofile -type f -delete
	rm gover.coverprofile
