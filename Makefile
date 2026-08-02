.PHONY: build test lint package clean

VERSION := $$(cat VERSION)

build:
	cd tui && go build -o ../build/codea ./cmd/codea

test:
	cd tui && go test ./...

lint:
	cd tui && golangci-lint run ./...

package: build
	./packaging/scripts/build-plugins.sh
	./packaging/scripts/collect-skills.sh
	./packaging/scripts/generate-manifest.sh
	./packaging/scripts/verify-checksum.sh

clean:
	rm -rf build/ packaging/staging/

phase0-gates:
	./scripts/run-phase0-gates.sh
