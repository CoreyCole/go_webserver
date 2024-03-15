# go webserver

### htmx + templ templates for HATEOAS

- [htmx](https://htmx.org/)
- [templ](https://templ.guide/)
- [HATEOAS](https://htmx.org/essays/hateoas/)

### react components for islands of interactivity

- Add react components to `react/components` directory
- build with `make`

### render markdown with code highlighting

- Add markdown files to the `public/md/` directory.
- Demo at [coreycc.com](https://coreycc.com/md/test.md)

### host bevy web assembly games

- Add bevy wasm bundled with trunk to the `public/games` directory

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
# copy systemd config (assumes pwd /home/ubuntu/go_webserver)
cp go_webserver.service /etc/systemd/system/go_webserver.service

# activate service
systemctl daemon-reload
service go_webserver start
service go_webserver status
```
