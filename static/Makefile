.PHONEY: build-deps build
export GEM_HOME=${PWD}/.gem

all: build

build-deps:
	which npm &>/dev/null || echo "ERROR: Please install npm" >&2
	which gem &>/dev/null || echo "ERROR: Please install gem" >&2
	npm install
	gem install sass

build: index.css index.js

index.css: index.sass
	./.gem/bin/sass --update index.sass

index.js: index.ts
	./node_modules/.bin/tsc --sourceMap --target ES2015 index.ts
