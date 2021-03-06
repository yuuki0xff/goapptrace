# Usage:
#   make            - Build the goapptrace.
#   make install    - Install goapptrace on your system.
#   make test       - Run linters and unit tests.
#   make build-deps - Install dependencies for build and test.
#   make help       - Print this message.
#
# Advanced:
#   make format          - Format source codes.
#   make check-formatted -
#   make release         -


.PHONEY: all build-deps build build-debug format check-formatted test release
SHELL=/bin/bash

all: build

help:
	@ sed '/^$$/Q; s/^# \?//;' Makefile

build-deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/alecthomas/gometalinter

	# gometalinterがforkしたリポジトリには最新のbug fixが反映されていない。
	# そのため、--no-vendored-lintersオプションを使用し、新しいlinterを使えるようにしている。
	# ただし、govetのリポジトリが無くなってしまったため、govetのインストールには失敗する。
	# workaroundとして、govetは個別にインストールをする。
	# See https://github.com/alecthomas/gometalinter/issues/436
	gometalinter -i -u --no-vendored-linters --debug || :
	go get -u github.com/golang/go/src/cmd/vet
	mv ${GOBIN}/vet ${GOBIN}/govet

	go get -t -u ./...

build:
	go build

build-debug:
	# Turn off optimization
	# See https://gist.github.com/tetsuok/3025333
	go build -gcflags '-N -l'

install:
	go install

format:
	goimports -w .
	go fmt ./...

check-formatted:
	@cmp <(goimports -e -d .) /dev/null &>/dev/null || ( \
		echo 'ERROR: Go import lines are NOT sorted.' && \
		echo '       Please execute command: "make format"' && \
		exit 1; \
	)
	@cmp <(gofmt -l -e .) /dev/null &>/dev/null || ( \
		echo 'ERROR: Source codes are NOT formatted.' && \
		echo '       Please execute command: "make format"' && \
		exit 1; \
	)

test: check-formatted
	gometalinter -D golint -D gocyclo -D maligned \
		--exclude "${GOROOT}|tracer/encoding/other/" \
		--enable-gc ./...
	go test ./...

release: test build
	git checkout master
	git merge --no-ff --no-edit develop
	version=$$(grep VERSION info/info.go |sed 's/.*"\(.*\)".*/\1/') && \
		git commit -m "Release v$${version}" && \
		git tag -s "v$${version}" -m "Relase v$${version}"
