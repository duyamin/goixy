build:
	go build -o goixy .

install:
	go build -ldflags "-s -w" -o goixy && cp goixy /usr/local/bin/
