GIT_HEAD = $(shell git rev-parse HEAD | head -c8)
LDFLAGS = -s -w -X superdaemon/system.Version=$(GIT_HEAD)

build:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -gcflags "all=-trimpath=$(pwd)" -o build/superdaemon_linux_amd64 -v superdaemon.go
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -gcflags "all=-trimpath=$(pwd)" -o build/superdaemon_linux_arm64 -v superdaemon.go

test:
	go vet ./...
	go test ./...

test-release:
	./scripts/test-release.sh

debug:
	go build -ldflags="-X superdaemon/system.Version=$(GIT_HEAD)"
	sudo ./superdaemon --debug --ignore-certificate-errors --config config.yml --pprof --pprof-block-rate 1

# Runs a remotly debuggable session for SuperDaemon allowing an IDE to connect and target
# different breakpoints.
rmdebug:
	go build -gcflags "all=-N -l" -ldflags="-X superdaemon/system.Version=$(GIT_HEAD)" -race
	sudo dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./superdaemon -- --debug --ignore-certificate-errors --config config.yml

cross-build: clean build

clean:
	rm -rf build/superdaemon_*

.PHONY: all build test test-release cross-build clean