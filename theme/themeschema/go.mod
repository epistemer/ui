module github.com/epistemer/ui/theme/themeschema

go 1.27.0

tool github.com/epistemer/cfgx/cmd/cfgx

// Until ui is tagged with ui/theme (v0.2.0), the module builds against the
// ui in this repository.
replace github.com/epistemer/ui => ../..

require (
	github.com/epistemer/cfgx v0.2.0
	github.com/epistemer/ui v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/a-h/templ v0.3.1118 // indirect
	github.com/epistemer/errors v0.1.0 // indirect
	github.com/epistemer/i18n v0.3.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.etcd.io/bbolt v1.5.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
