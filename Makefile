BINARY := ytt

.PHONY: build install clean

build:
	go build -o $(BINARY) ./cmd/ytt

install:
	go install ./cmd/ytt

clean:
	rm -f $(BINARY)
