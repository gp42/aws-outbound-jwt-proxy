BINARY  := aws-outbound-jwt-proxy
DIST    := dist
VERSION ?= dev
VERSION_PKG     := github.com/gp42/aws-outbound-jwt-proxy/internal/version
LDFLAGS_VERSION := -X $(VERSION_PKG).Version=$(VERSION)
LDFLAGS         := -s -w $(LDFLAGS_VERSION)

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build build-debug test fmt vet tidy run clean build-all install-hooks uninstall-hooks $(PLATFORMS)

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) .

# Unstripped build for local debugging with dlv / full symbol tables.
build-debug:
	go build -ldflags="$(LDFLAGS_VERSION)" -o bin/$(BINARY) .

build-all: $(PLATFORMS)

$(PLATFORMS):
	@os=$(word 1,$(subst /, ,$@)); \
	arch=$(word 2,$(subst /, ,$@)); \
	ext=$$( [ $$os = windows ] && echo .exe || echo ); \
	out=$(DIST)/$(BINARY)-$$os-$$arch$$ext; \
	echo "building $$out"; \
	GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $$out .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

run:
	go run .

clean:
	rm -rf bin $(DIST)

# Install repo-tracked git hooks (Conventional Commits commit-msg hook).
install-hooks:
	git config core.hooksPath hack/hooks
	@echo "hooks installed: core.hooksPath=hack/hooks"

uninstall-hooks:
	-git config --unset core.hooksPath
	@echo "hooks uninstalled"
