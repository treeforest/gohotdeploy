.PHONY: statik
statik:
	statik -src=public -dest=.

.PHONY: build
build:
	go build -ldflags="-s -w" -o ./bin/gohotdeploy .