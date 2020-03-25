static/bundle.js: $(shell find ./client)
	./node_modules/.bin/rollup -c rollup.config.js

deploy: static/bundle.js
	netlify deploy --prod --dir=static
