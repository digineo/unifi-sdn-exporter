.PHONY: unifi-sdn-exporter
unifi-sdn-exporter:
	go build -ldflags="-s -w" -trimpath -o $@ main.go

.PHONY: dev
dev: unifi-sdn-exporter config.toml
	./$< --web.config=config.toml

.PHONY: release
release:
	goreleaser release --clean --skip sign,publish
