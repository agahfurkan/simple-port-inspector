BINARY=port-inspector
GOFLAGS=-ldflags="-s -w"

.PHONY: build run clean install

build:
	go build $(GOFLAGS) -o $(BINARY) ./main.go

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

dev:
	go run ./main.go
