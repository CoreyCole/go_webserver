# go markdown webserver with code highlighting
- Add markdown files to the `www/md/` directory.
- Demo at [coreycc.com](https://coreycc.com/md/test.md)

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

Build binary
```bash
go build -o bin/go_webserver main.go
```

Systemd setup
```bash
# copy config as root
sudo cp go_webserver.service /etc/systemd/system/go_webserver.service

# activate service
systemctl daemon-reload
service go_webserver start
service go_webserver status
```
