.PHONY: run build test vet clean

run:
	go run ./cmd/dcportal/

build:
	go build -o bin/dcportal ./cmd/dcportal/

test:
	go test ./... -v

test-cover:
	go test ./... -cover

vet:
	go vet ./...

clean:
	rm -rf bin/ data/
