build:
	go build -o goixy

install:
	go build -ldflags "-s -w" -o goixy && cp goixy /usr/local/bin/

run:
	nohup /usr/local/bin/goixy --verbose > /tmp/goixy.log &
