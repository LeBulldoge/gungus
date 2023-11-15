VERSION := $(shell git describe --tags --long)
BUILD   := $(shell date -u +%Y%m%d_%H%M%S)

BUILD_FLAGS = "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"

all: clean build.all

clean:
	rm -rf build/

build: gungus

build.all: gungus.linux.amd64 gungus.linux.arm64 gungus.darwin.arm64

gungus:
	go build -o build/$@ -ldflags ${BUILD_FLAGS} -trimpath

gungus.linux.amd64:
	GOOS=linux GOARCH=amd64 go build -o build/$@ -ldflags ${BUILD_FLAGS} -trimpath
gungus.linux.arm64:
	GOOS=linux GOARCH=arm64 go build -o build/$@ -ldflags ${BUILD_FLAGS} -trimpath
gungus.darwin.arm64:
	GOOS=darwin GOARCH=arm64 go build -o build/$@ -ldflags ${BUILD_FLAGS} -trimpath

install:
	go install -ldflags ${BUILD_FLAGS} -trimpath

docker:
	podman build -f builder.Dockerfile --build-arg='VERSION=${VERSION}' --build-arg='BUILD=${BUILD}'

publish:
	podman build --platform=linux/arm64,linux/amd64 --manifest ghcr.io/lebulldoge/gungus:${VERSION} -f builder.Dockerfile --build-arg='VERSION=${VERSION}' --build-arg='BUILD=${BUILD}'
	podman manifest push ghcr.io/lebulldoge/gungus:${VERSION}
	podman manifest rm ghcr.io/lebulldoge/gungus:${VERSION}
	podman manifest create --all ghcr.io/lebulldoge/gungus:latest ghcr.io/lebulldoge/gungus:${VERSION}
	podman manifest push ghcr.io/lebulldoge/gungus:latest
	podman manifest rm ghcr.io/lebulldoge/gungus:latest
