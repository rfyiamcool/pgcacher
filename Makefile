all: build

build:
	@GOPROXY=https://goproxy.cn go mod tidy
	@GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn go build .
