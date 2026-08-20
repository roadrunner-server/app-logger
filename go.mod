module github.com/roadrunner-server/app-logger/v6

go 1.26

toolchain go1.26.6

require (
	github.com/roadrunner-server/api-go/v6 v6.0.0-beta.14
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
