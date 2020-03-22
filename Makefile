all: build

.PHONY: build
build:
	go build -o bin/qumomf cmd/qumomf/main.go

.PHONY: run
run: build
	bin/qumomf -config=example/qumomf.yaml

.PHONY: run_docker
run_docker:
	docker-compose -f example/docker-compose.yml up -d

.PHONY: down_docker
down_docker:
	docker-compose -f example/docker-compose.yml down -v --rmi local --remove-orphans

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run -v ./...

.PHONY: test
test:
	go test -count=1 ./...

.PHONY: integration_test
integration_test:
	cd example && go test -run Test_Router_AddAndCheckKey -count=1 -v -tags=integration ./...
	docker-compose -f example/docker-compose.yml stop storage_1_m storage_2_m
	cd example && go test -run Test_Router_AddAndCheckKey -count=1 -v -tags=integration ./...
