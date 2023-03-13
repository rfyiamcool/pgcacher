all: build

build:
	@env GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn go build .
