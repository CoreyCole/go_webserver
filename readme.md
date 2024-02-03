# go markdown webserver with code highlighting
- Web Assembly game written in Rust at [http://localhost:3000/bevy](http://localhost:3000/bevy)
- Rendered markdown at [http://localhost:3000/md/test.md](http://localhost:3000/md/test.md).
- Add markdown files to the `www/md/` directory.

Run the server. 
```bash
go run main.go
```

Watch & generate [templ template](https://templ.guide/quick-start/installation) views
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
