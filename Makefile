.PHONY: dev server web build test verify server-format server-vet server-test server-race vulncheck docker-dev docker

dev:
	@make -j 2 server web

server:
	go run ./server/cmd/tripleagent

web:
	bun run --cwd web dev

build:
	bun run --cwd web build

server-format:
	@files="$$(gofmt -l server)"; if [ -n "$$files" ]; then echo "Go files need gofmt:"; echo "$$files"; exit 1; fi

server-vet:
	go vet ./server/...

server-test:
	go test ./server/...

server-race:
	go test -race ./server/...

vulncheck:
	govulncheck ./server/...

test: server-test
	bun --cwd web test

verify: server-format server-vet server-test
	bun run --cwd web verify

docker-dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

docker:
	docker compose up --build
