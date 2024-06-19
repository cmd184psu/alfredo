module github.com/cmd184psu/alfredo

go 1.21

require (
	github.com/aws/aws-sdk-go v1.46.1
	github.com/cmd184psu/fs-tools/fstools-gomod v0.0.0-20220902171344-d0f349b98770
	github.com/pkg/sftp v1.13.6
	golang.org/x/crypto v0.14.0
)

require (
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/ini.v1 v1.67.0
)

replace github.com/cmd184psu/alfredo => ./

replace github.com/cmd184psu/alfredo/exec => ./exec
