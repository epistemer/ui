module github.com/epistemer/ui/examples/shop

go 1.27.0

replace github.com/epistemer/ui => ../..

require (
	github.com/epistemer/errors v0.1.0
	github.com/epistemer/i18n v0.3.1
	github.com/epistemer/models v0.3.0
	github.com/epistemer/ui v0.0.0-00010101000000-000000000000
)

require (
	go.etcd.io/bbolt v1.5.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
