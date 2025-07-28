# MeetingBar Makefile

BINARY_NAME=meetingbar
BUILD_DIR=build
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: all build clean install test lint fmt deps

all: deps build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build the application
build:
	mkdir -p ${BUILD_DIR}
	CGO_ENABLED=1 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .

# Build for different architectures
build-linux-amd64:
	mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .

build-linux-arm64:
	mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 .

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf ${BUILD_DIR}
	go clean

# Install system-wide (requires sudo)
install: build
	sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	mkdir -p ~/.local/share/applications
	cp build/linux/meetingbar.desktop ~/.local/share/applications/
	update-desktop-database ~/.local/share/applications

# Uninstall
uninstall:
	sudo rm -f /usr/local/bin/${BINARY_NAME}
	rm -f ~/.local/share/applications/meetingbar.desktop
	update-desktop-database ~/.local/share/applications

# Create installation package
package: build
	mkdir -p ${BUILD_DIR}/package/meetingbar-${VERSION}
	cp ${BUILD_DIR}/${BINARY_NAME} ${BUILD_DIR}/package/meetingbar-${VERSION}/
	cp README.md ${BUILD_DIR}/package/meetingbar-${VERSION}/
	cp build/linux/install.sh ${BUILD_DIR}/package/meetingbar-${VERSION}/
	cp build/linux/meetingbar.desktop ${BUILD_DIR}/package/meetingbar-${VERSION}/
	cd ${BUILD_DIR}/package && tar -czf meetingbar-${VERSION}-linux.tar.gz meetingbar-${VERSION}

# Development - run with debug logging
dev: build
	DEBUG=1 ./${BUILD_DIR}/${BINARY_NAME}