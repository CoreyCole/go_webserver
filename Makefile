build:
	@npx tailwindcss -i webserver/view/css/index.css -o public/build.css
	@templ generate view
	@go build -o bin/go_webserver main.go
	@node esbuild.js

watch:
	air

run: build
	@./bin/go_webserver

install:
	@go install github.com/cosmtrek/air@latest
	@go install github.com/a-h/templ/cmd/templ@latest
	@go get ./...
	@go mod tidy
	@go mod download
	@npm install -g pnpm@latest
	@pnpm install


up: ## Database migration up
	@go run cmd/migrate/main.go up

reset:
	@go run cmd/reset/main.go up

down: ## Database migration down
	@go run cmd/migrate/main.go down

migration: ## Migrations against the database
	@migrate create -ext sql -dir cmd/migrate/migrations $(filter-out $@,$(MAKECMDGOALS))

seed:
	@go run cmd/seed/main.go
