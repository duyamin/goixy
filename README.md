# Goixy

An HTTP/SOCKS5 Proxy in Go

![https://github.com/mitnk/goixy/blob/master/howitworks.png](https://github.com/mitnk/goixy/blob/master/howitworks.png)

## install

You can download [pre-built binaries here](https://github.com/mitnk/goixy/releases).

Or build it with Go environment yourself:

```
$ go get -u github.com/mitnk/goixy
```

## usage

### config

```
$ cat ~/.goixy/config.json
{
    "Host": "1.2.3.4",
    "Port": "5678",
    "Key": "your-lightsocks-secret-key",
    "WhiteList": [
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
use `Host:Port` proxy. If `-withdirect` is set, only `WhiteList` connections
use `Host:Port` proxy, other traffic use `DirectHost:DirectPort` proxy.

### run it

```
$ goixy
[2017-06-18 14:58:36][0] goixy v1.6.7 without Direct Porxy
[2017-06-18 14:58:36][0] listen on port: 127.0.0.1:1080
```

### see its help page

```
$ goixy -h
Usage of goixy v1.6.7
goixy [flags]
  -host string
        host (default "127.0.0.1")
  -port string
        port (default "1080")
  -s int
        for each many seconds print reports (default 600)
  -v    verbose
  -vv
        very verbose
  -withdirect
        Use Direct proxy (for HTTP Proxy only)
```


NOTE: currently `-withdirect` only supports HTTP Proxy. (socksv5 proxy seems
always got IP instead of domains). So even set `-withdirect`, accesses with
Socks Porxy (i.e. `curl -x socks5://...`) will always use `Host:Port` proxy.

## lightsocks

[https://github.com/mitnk/lightsocks](https://github.com/mitnk/lightsocks)
