all: build

.PHONY: build
build:
	go build -o bin/qumomf cmd/qumomf/main.go

.PHONY: run
run: build
	bin/qumomf -config=example/qumomf.yaml

.PHONY: env_up
env_up:
	docker-compose -f example/docker-compose.yml up -d
	sleep 1
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

.PHONY: run_unit_tests
run_unit_tests:
	go test -count=1 -v ./...

.PHONY: run_integration_tests
run_integration_tests:
	go test -count=1 -v -tags=integration ./...

.PHONY: run_tests
run_tests: env_up run_integration_tests

.PHONY: run_failover_test
run_failover_test:
	cd example && go test -run Test_Router_AddAndCheckKey -count=1 -v -tags=failover ./...
	docker-compose -f example/docker-compose.yml stop storage_1_m storage_2_m
	cd example && go test -run Test_Router_AddAndCheckKey -count=1 -v -tags=failover ./...
