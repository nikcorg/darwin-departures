LD_FLAGS := -X \"main.DarwinToken=${DARWIN_TOKEN}\"
BIN_NAME := departures

.PHONY: build
build:
	mkdir -p bin && gotip build -ldflags="$(LD_FLAGS)" -o bin/$(BIN_NAME)