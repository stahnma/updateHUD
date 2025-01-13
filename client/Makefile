

fmt:
	go fmt ./...

build: fmt
	go build -o client .


linux: fmt
	GOOS=linux GOARCH=amd64 go build -o client.linux.amd64 .
	GOOS=linux GOARCH=arm64 go build -o client.linux.arm64 .
	GOOS=linux GOARCH=arm go build -o client.linux.arm .

clean:
	rm -f  client client.linux.amd64
