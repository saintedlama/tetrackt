.PHONY: build lint test run clean

BINARY := tetrackt

build:
	go build -o $(BINARY) .

lint:
	go vet ./...

test:
	go test ./...

run:
	go run .

clean:
	go clean
	rm -f $(BINARY)
