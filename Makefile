

NAME=mqttfun

build: fmt tidy
	go build .

tidy:
	go mod tidy

fmt:
	go fmt *.go

run: fmt
	go run *.go

install: build
#	sudo install -p -m755 slack_channel_list /usr/local/bin
#
arm: fmt
	GOOS=linux GOARCH=arm go build .
	scp $(NAME) root@fred:

rpm: fmt
	GOOS=linux GOARCH=amd64 go build .
	scp $(NAME) root@giga2:

clean:
	rm -f $(NAME) mqttfun
	go clean -modcache

json:
	cat message | jq > m ; mv m message


iter:
	go build -o upc client/main.go 
