# Goixy

An HTTP/SOCKS5 Proxy in Go

## install & usage

```
$ go get github.com/mitnk/goixy

$ goixy -h
Usage of goixy v1.6.2
goixy [flags]
  -host string
        host (default "127.0.0.1")
  -port string
        port (default "1080")
  -v    verbose
  -vv
        very verbose
  -withdirect
        Use Direct proxy
```

## config

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

You need to run [lightsocks](https://github.com/mitnk/lightsocks) on
`1.2.3.4:5678` at least. And also run on `127.0.0.1:12345` if you use
`-withdirect`.

Goixy default does not use direct proxy, means that all connections will
use `Host:Port` proxy. If `-withdirect` is set, only WhiteList connections
using `Host:Port` proxy, other traffic use `DirectHost:DirectPort` proxy.

NOTE: currently only HTTP Proxy support `-withdirect`. Because socksv5 proxy
seems got IP instead of domains.

## how it works

![https://github.com/mitnk/goixy/blob/master/howitworks.png](https://github.com/mitnk/goixy/blob/master/howitworks.png)

## lightsocks

[https://github.com/mitnk/lightsocks](https://github.com/mitnk/lightsocks)
