COMPOSE = docker compose run --rm dev

.PHONY: build lint test test-unit test-integration godoc shell

build:
	docker compose build

lint:
	$(COMPOSE) golangci-lint run ./...

test:
	$(COMPOSE) gotestsum --format testdox -- -coverprofile=coverage.out ./...

test-unit:
	$(COMPOSE) gotestsum --format testdox -- -short -coverprofile=coverage.out ./...

test-integration:
	$(COMPOSE) gotestsum --format testdox -- -run Integration -count=1 ./...

godoc:
	docker compose run --rm -p 6060:6060 dev godoc -http=:6060

shell:
	$(COMPOSE) bash
