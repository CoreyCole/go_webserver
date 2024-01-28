# go markdown webserver with code highlighting
Rendered markdown at [`http://localhost:3000/md/test.md`](http://localhost:3000/md/test.md). Add markdown files to the `md/` directory.

Run the server. 
```bash
go run main.go
```

Watch & generate templ template views
```bash
templ generate --watch
```

Hot reload local dev server with air
```bash
air
```

Run all Tests
```bash
go test ./...
```
