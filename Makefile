# Shortcuts for the course. Run `make help` to see them.
.DEFAULT_GOAL := help
.PHONY: help list run all test race cover bench fuzz fmt vet lint check api tidy clean

help: ## show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Examples:"
	@echo "    make run L=14        run lesson 14"
	@echo "    make run L=channels  run a lesson by name"

list: ## list all lessons
	@go run .

L ?=
run: ## run one lesson: make run L=14
	@test -n "$(L)" || { echo "usage: make run L=<number|name>"; exit 1; }
	@go run . $(L)

all: ## run every lesson in order
	@go run . all

test: ## run all tests
	go test ./...

race: ## run all tests with the race detector
	go test -race -count=1 ./...

cover: ## test coverage report (opens coverage.html)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.html"

bench: ## run benchmarks with allocation counts
	go test -bench=. -benchmem -run='^$$' ./...

fuzz: ## fuzz the lesson 24 example for 15s
	go test -fuzz=FuzzAdd -fuzztime=15s ./lessons/24_testing/cart

fmt: ## format every file
	gofmt -w .

vet: ## run go vet
	go vet ./...

lint: ## run golangci-lint if installed
	@command -v golangci-lint >/dev/null \
		&& golangci-lint run \
		|| echo "golangci-lint not installed: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

check: fmt vet race ## format, vet and test with -race

api: ## run the capstone service
	go run ./projects/taskapi/cmd/api

tidy: ## sync go.mod
	go mod tidy

clean: ## remove build artifacts
	go clean -cache -testcache
	rm -f coverage.out coverage.html
