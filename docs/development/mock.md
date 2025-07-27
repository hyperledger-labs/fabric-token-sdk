# Generation Mock Files

`app/services/info/mock` directory contains mock auto-generated files.
The generation directives can be found, for example at `app/services/info/dbs.go`:
```
//go:generate counterfeiter -o mock/userdb.go -fake-name UserDB . UserDB
```
To regenerate all mock files, go to `app/services/info/` and run `go generate`.
