build:
	goreleaser build --id $(shell go env GOOS) --single-target --snapshot --clean
darwin:
	goreleaser build --id darwin --snapshot --clean
linux:
	goreleaser build --id linux --snapshot --cleant
snapshot:
	goreleaser release --snapshot --clean
tag:
	git tag $(shell svu next)
	git push --tags
release:
	goreleaser --clean

qa:
	golangci-lint run -D errcheck

consul-dev:
	consul agent -dev
consul-config:
	consul config write scripts/failover.hcl
	consul config write scripts/redirect.hcl
nomad-dev:
	sudo nomad agent -dev \
      -bind 0.0.0.0 \
      -network-interface='{{ GetDefaultInterfaces | attr "name" }}'
