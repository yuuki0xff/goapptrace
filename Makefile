.PHONEY: all build-deps build build-debug format check-formatted test release
SHELL=/bin/bash

all: build

build-deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/alecthomas/gometalinter
	gometalinter -u
	go get -t -u ./...

build:
	go build

build-debug:
	# Turn off optimization
	# See https://gist.github.com/tetsuok/3025333
	go build -gcflags '-N -l'

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
	gometalinter -D golint -D gocyclo --exclude "${GOROOT}" ./...
	go test ./...

release: test build
	git checkout master
	git merge --no-ff --no-edit develop
	version=$$(grep VERSION info/info.go |sed 's/.*"\(.*\)".*/\1/') && \
		git commit -m "Release v$${version}" && \
		git tag -s "v$${version}" -m "Relase v$${version}"
