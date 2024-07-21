build:
	@go build -o bin/budget-tracker-api

run: build
	@./bin/budget-tracker-api

test:
	@go test -v ./...


run-dev:
	ENV='dev' air