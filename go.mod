module github.com/robertkonga/yekonga-server-go


go 1.24.0

toolchain go1.24.5

require (
	github.com/kardianos/service v1.2.4 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/tiendc/go-deepcopy v1.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/excelize/v2 v2.10.0 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	golang.org/x/text v0.30.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/go-cmp v0.6.0
)

require (
	filippo.io/edwards25519 v1.1.0
	github.com/golang/snappy v0.0.4
	github.com/klauspost/compress v1.16.7
	github.com/xdg-go/scram v1.1.2
	github.com/xdg-go/stringprep v1.0.4
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78
	golang.org/x/crypto v0.43.0
	golang.org/x/net v0.46.0
	golang.org/x/sync v0.17.0
	golang.org/x/sys v0.37.0
)

replace golang.org/x/net/http2 => golang.org/x/net/http2 v0.23.0 // GODRIVER-3225
