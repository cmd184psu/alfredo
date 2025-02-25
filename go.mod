module github.com/cmd184psu/alfredo

go 1.21

require (
	github.com/aws/aws-sdk-go v1.46.1
	github.com/pkg/sftp v1.13.6
	golang.org/x/crypto v0.14.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-chi/chi v1.5.5
	github.com/go-chi/chi/v5 v5.1.0
	github.com/gorilla/websocket v1.5.3
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/ini.v1 v1.67.0
)

replace github.com/cmd184psu/alfredo => ./

replace github.com/cmd184psu/alfredo/exec => ./exec
