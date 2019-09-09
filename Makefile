lnurl-playground: $(shell find . -name "*.go") bindata.go
	go build

static/bundle.js: $(shell find ./client)
	npm run build

bindata.go: static/bundle.js static/index.html static/global.css
	go-bindata -o bindata.go static/...
