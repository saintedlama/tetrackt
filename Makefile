.PHONY: build lint test run clean compress

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

compress:
	cd persistence/akwf && tar -czf akwf.tar.gz *.json
