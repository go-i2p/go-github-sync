module github.com/go-i2p/go-github-sync

go 1.26.0

require (
	github.com/google/go-github/v61 v61.0.0
	github.com/spf13/cobra v1.10.2
	go.uber.org/zap v1.28.0
	golang.org/x/oauth2 v0.37.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

retract (
	v0.1.59999
	v0.1.5999
	v0.1.599
)
