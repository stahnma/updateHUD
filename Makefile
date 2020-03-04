

fmt:
	go fmt *.go

build: fmt
	go build *.go

run: fmt
	go run *.go

install: build
#	sudo install -p -m755 slack_channel_list /usr/local/bin
#
arm: fmt
	GOOS=linux GOARCH=arm go build reporter.go
	scp reporter root@fred:

rpm: fmt
	GOOS=linux go build reporter.go
	scp reporter root@spike:

clean:
	rm -f reporter mqttfun
