APP-BIN := dist/$(shell basename $(shell pwd))

.PHONY: build
build:
	goreleaser build --id $(shell go env GOOS) --single-target --snapshot --clean -o ${APP-BIN}
.PHONY: darwin
darwin:
	goreleaser build --id darwin --snapshot --clean
.PHONY: linux
linux:
	goreleaser build --id linux --snapshot --clean
.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --clean
.PHONY: tag
tag:
	git tag $(shell svu next)
	git push --tags
.PHONY: release
release: tag
	goreleaser --clean

.PHONY: watch
watch:
	gotestsum --watch --format testname
.PHONY: lint
lint:
	pre-commit run --files $(shell git ls-files -m)
.PHONY: test
test:
	gotestsum --format testname
.PHONY: qa
qa: lint test

.PHONY: run
run: ## Run binary.
	./${APP-BIN} server
.PHONY: fresh
fresh: build run
.PHONY: consul-dev
consul-dev:
	consul agent -dev
.PHONY: consul-config
consul-config:
	consul config write scripts/failover.hcl
	consul config write scripts/redirect.hcl
.PHONY: nomad-dev
nomad-dev:
	nomad agent -dev -bind 0.0.0.0
	sudo nomad agent -dev \
	-bind 0.0.0.0 \
	-network-interface='{{ GetDefaultInterfaces | attr "name" }}'
