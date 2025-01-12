

fmt:
	go fmt ./...

build: fmt
	go build -o client .


linux: fmt
	GOOS=linux GOARCH=amd64 go build -o client.linux.amd64 .

clean:
	rm -f  client client.linux.amd64
