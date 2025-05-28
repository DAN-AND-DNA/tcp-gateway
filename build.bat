SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
SET GOPROXY=https://goproxy.cn,direct
go build -o gateway -v ./cmd/gateway