module github.com/cmd184psu/alfredo

go 1.26.1

require (
	github.com/aws/aws-sdk-go v1.55.8
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/pkg/sftp v1.13.10
	golang.org/x/crypto v0.49.0
	golang.org/x/term v0.41.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/gorilla/websocket v1.5.3
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/stretchr/testify v1.11.1
	golang.org/x/sys v0.42.0
	gopkg.in/ini.v1 v1.67.1
)

replace github.com/cmd184psu/alfredo => ./

replace github.com/cmd184psu/alfredo/exec => ./exec
