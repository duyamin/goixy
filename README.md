# Goixy

An HTTP/SOCKS5 Proxy, written in Go.

![https://github.com/mitnk/goixy/blob/master/howitworks.png](https://github.com/mitnk/goixy/blob/master/howitworks.png)

## install

You can download [pre-built binaries here](https://github.com/mitnk/goixy/releases).

Or build it with Go environment yourself:

```
$ go get -u github.com/mitnk/goixy
```

## usage

First, you need to create a config file for goixy. It locates at
`~/.goixy/config.json`, and looks like this:

```
$ cat ~/.goixy/config.json
{
    "Host": "1.2.3.4",
    "Port": "5678",
    "Key": "your-lightsocks-secret-key",
    "DomainList": [
        "\\.google.*",
        ".*facebook\\.com"
    ],
    "DirectHost": "127.0.0.1",
    "DirectPort": "12345",
    "DirectKey": ""
}
```

(If `DirectKey` is not set or empty, `Key` will be used)

You need to run [lightsocks](https://github.com/mitnk/lightsocks) on
`1.2.3.4:5678`. And also need to run on `127.0.0.1:12345` if you use
`-withdirect`.

Goixy default does not use direct proxy, meaning all connections will
use `Host:Port` proxy. If `-withdirect` is set, only `Domains` connections
use `Host:Port` proxy, other traffic use `DirectHost:DirectPort` proxy.

### run it

```
$ goixy
[2017-06-18 14:58:36][0] goixy v1.8.0 without Direct Porxy
[2017-06-18 14:58:36][0] listen on port: 127.0.0.1:1080
```

Now you can test it with curl:

```bash
$ curl -L -x 127.0.0.1:1080 hugo.wang/http/ip/
1.2.3.4  # output should be the IP of host on which lightsocks is running
$ curl -L hugo.wang/http/ip/
111.112.113.114  # should be you local public IP
```

### see its help page

```
$ goixy -h

Usage of goixy v1.8.0
goixy [FLAGS]

  -db int
        Redis DB index (default 7)
  -host string
        host (default "127.0.0.1")
  -port string
        port (default "1080")
  -t int
        time out on connections in seconds (default 3600)
  -v    verbose, print some debug info
  -vv
        very verbose, more debug info
  -wbl
        Use balcklist (for HTTP only)
  -wd
        Use Direct proxy (for HTTP Porxy only)
```

NOTE: currently `-wd`, `-wbl` only supports HTTP Proxy. Even set
`-withdirect`, accesses with Socks Porxy (i.e. `curl -x socks5://...`)
will always use `Host:Port` proxy.

## Blacklist Operations

```
$ py scripts/add-item.py -h
$ py scripts/list-items.py -h
$ py scripts/import-items.py -h

# demo
$ py scripts/import-items.py -f blacklist.txt --list black
```

## lightsocks

[https://github.com/mitnk/lightsocks](https://github.com/mitnk/lightsocks)
