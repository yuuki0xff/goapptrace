.PHONEY: build-deps build release

all: build

build-deps:
	$(MAKE) -C static build-deps
	go get

build:
	$(MAKE) -C static build
	go build

release: build
	git checkout master
	git merge --no-ff --no-edit develop
	git add --force static/*.js static/*.css
	version=$$(grep VERSION info/info.go |sed 's/.*"\(.*\)".*/\1/') && \
		git commit -m "Release v$${version}" && \
		git tag -s "v$${version}" -m "Relase v$${version}"
