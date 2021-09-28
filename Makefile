lnurl-playground: $(shell find . -name "*.go") static/bundle.js static/index.html static/global.css static/codec/bundle.js
	go build -ldflags="-s -w"

static/bundle.js: $(shell find ./client)
	./node_modules/.bin/rollup -c rollup.config.js

codec/static/bundle.js: $(shell find ./codec/client)
	cd codec && make

static/codec/bundle.js: codec/static/bundle.js codec/static/index.html codec/static/bundle.css
	rm -rf static/codec
	cp -r codec/static static/codec

deploy: lnurl-playground
	ssh root@turgot 'systemctl stop lnurl'
	scp lnurl-playground turgot:lnurl-playground/lnurl-playground
	ssh root@turgot 'systemctl start lnurl'
