.PHONEY: build-deps build format check-formatted test release
SHELL=/bin/bash

all: build

build-deps:
	$(MAKE) -C static build-deps
	go get

build:
	$(MAKE) -C static build
	go build

format:
	go fmt ./...

check-formatted:
	@cmp <(gofmt -l -e .) /dev/null &>/dev/null || ( \
		echo 'ERROR: Source codes are NOT formatted.' && \
		echo '       Please execute command: "make format"' && \
		exit 1; \
	)

test: check-formatted
	go test ./...

release: test build
	git checkout master
	git merge --no-ff --no-edit develop
	git add --force static/*.js static/*.css
	version=$$(grep VERSION info/info.go |sed 's/.*"\(.*\)".*/\1/') && \
		git commit -m "Release v$${version}" && \
		git tag -s "v$${version}" -m "Relase v$${version}"
