module resty-example

go 1.24.3

require (
	github.com/go-resty/resty/v2 v2.17.1
	smartlog v0.0.0-00010101000000-000000000000
)

replace smartlog => ../..

require (
	github.com/DeRuina/timberjack v1.3.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/net v0.43.0 // indirect
)
