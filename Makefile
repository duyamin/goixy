build:
	go build -o goixy .

all:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o goixy-mac
	GOOS=windows GOARCH=386 go build -ldflags "-s -w" -o goixy32.exe
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o goixy64.exe
	GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o goixy-linux-32
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o goixy-linux-64
	GOOS=linux GOARCH=arm go build -ldflags "-s -w" -o goixy-arm-32
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o goixy-arm-64

install:
	go build -ldflags "-s -w" -o goixy && cp goixy /usr/local/bin/

clean:
	rm -f goixy goixy-* *.exe
