module github.com/roadrunner-server/app-logger/v6

go 1.27

toolchain go1.27.2

require (
	github.com/roadrunner-server/api-go/v6 v6.1.0
	github.com/stretchr/testify v1.12.1
)

exclude (
	github.com/spf13/viper v1.18.0
	github.com/spf13/viper v1.18.1
)

require (
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
