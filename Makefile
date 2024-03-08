build:
	@go build -o bin/budget-tracker-api

run: build
	@./bin/budget-tracker-api

test:
	@go test -v ./...