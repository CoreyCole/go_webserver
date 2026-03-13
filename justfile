default:
    @just --list

build:
    sqlc generate
    templ generate -path webserver/view
    pnpm exec tailwindcss -i webserver/view/css/index.css -o public/build.css --content "./webserver/**/*.go" --content "./webserver/**/*.templ"
    go build -o bin/go_webserver main.go

watch:
    air

run: build
    ./bin/go_webserver

install:
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    go install github.com/air-verse/air@latest
    go install github.com/a-h/templ/cmd/templ@latest
    go get ./...
    go mod tidy
    go mod download
    npm install -g pnpm@latest
    pnpm install

# Verify everything compiles (fmt + generate + lint + build)
check:
    ./scripts/check.sh

sync-thoughts:
    git add thoughts/ && git commit -m "sync thoughts" || echo "nothing to sync"
