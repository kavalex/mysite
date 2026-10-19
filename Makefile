GOFLAGS := -trimpath -ldflags="-s -w"
BIN     := mysite

.PHONY: build test lint vet run clean build-all deps modernize

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o dist/$(BIN) .

test:
	go test -race -coverprofile=cover.out ./...
	go tool cover -func=cover.out | tail -1

lint:
	golangci-lint run --timeout=5m
	govulncheck ./...

vet:
	go vet ./...
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needs to run on:"; echo "$$out"; exit 1; fi

modernize:
	go fix ./...

run: build
	./dist/$(BIN)

build-all:
	@mkdir -p dist
	@for target in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do \
		os=$${target%/*}; arch=$${target#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "Building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(GOFLAGS) \
			-o "dist/$(BIN)-$$os-$$arch$$ext" . || exit 1; \
	done

clean:
	rm -rf dist cover.out

deps:
	go mod download
	go mod tidy
