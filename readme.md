# go markdown webserver with code highlighting
- Add markdown files to the `static/md/` directory.
- Demo at [coreycc.com](https://coreycc.com/md/test.md)

Install dependencies.
```
make install
```

Run the server. 
```bash
make run
```

Watch for local file changes.
```
make watch
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
