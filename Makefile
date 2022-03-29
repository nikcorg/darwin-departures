LD_FLAGS := -X \"main.DarwinToken=${DARWIN_TOKEN}\"
BIN_NAME := departures

.PHONY: build
build: dirs
	go build -ldflags="$(LD_FLAGS)" -o bin/$(BIN_NAME)
	GOOS=linux go build -ldflags="$(LD_FLAGS)" -o bin/$(BIN_NAME)-linux
	GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="$(LD_FLAGS)" -o bin/$(BIN_NAME)-linux-arm5

.PHONY: dirs
dirs:
	mkdir -p bin

.PHONY: clean
clean:
	rm -rf bin