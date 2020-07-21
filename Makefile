.PHONY: run

MAIN_FILE="./cmd/qa-discussion/main.go"

run-server:
	@echo qa-discussion is running..
	go run ${MAIN_FILE}

dev-install:
	go get -u github.com/pressly/goose/cmd/goose
	go get -u github.com/go-delve/delve

debug-server:
	dlv debug ${MAIN_FILE}

migrate:
	./db/migrate.sh up

migrate-reset:
	./db/migrate.sh reset

migrate-test:
	./db/migrate_test.sh up

migrate-test-reset:
	./db/migrate_test.sh reset

build-linux:
	env GOOS=linux GOARCH=amd64 go build ${MAIN_FILE}

build-darwin:
	env GOOS=darwin GOARCH=amd64 go build ${MAIN_FILE}

docker-build:
	docker build -t qa-discussion .
