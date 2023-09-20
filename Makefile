all: clean build.all

clean:
	rm -rf build/

build: gungus

build.all: gungus.linux.amd64 gungus.linux.arm64 gungus.darwin.arm64

gungus:
	go build -o build/$@ -ldflags "-w -s" -trimpath

gungus.linux.amd64:
	GOOS=linux GOARCH=amd64 go build -o build/$@ -ldflags "-w -s" -trimpath
gungus.linux.arm64:
	GOOS=linux GOARCH=arm64 go build -o build/$@ -ldflags "-w -s" -trimpath
gungus.darwin.arm64:
	GOOS=darwin GOARCH=arm64 go build -o build/$@ -ldflags "-w -s" -trimpath

install:
	go install -ldflags="-s -w" -trimpath
