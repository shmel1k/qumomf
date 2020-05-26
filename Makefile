BINARY=qumomf
GIT_SUMMARY=`git describe --tags --dirty --always`
BUILD_DATE=`date +%FT%T%z`
LDFLAGS=-ldflags "-w -s -X main.Version=${GIT_SUMMARY} -X main.BuildDate=${BUILD_DATE}"

all: build

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/${BINARY} cmd/qumomf/main.go

.PHONY: run
run: build
	bin/qumomf -config=example/qumomf.yml

.PHONY: env_up
env_up:
	docker-compose -f example/docker-compose.yml up -d
	sleep 2
	docker-compose -f example/docker-compose.yml ps

.PHONY: env_down
env_down:
	docker-compose -f example/docker-compose.yml down -v --rmi local --remove-orphans

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run -v ./...

.PHONY: run_short_tests
run_short_tests:
	go test -count=1 -v -short ./...

.PHONY: run_tests
run_tests: env_up
	go test -count=1 -v -race ./...
