# Goixy

An HTTP/SOCKS5 Proxy in Go

## install & usage

```
$ go get github.com/mitnk/goixy

$ goixy -h
Usage of goixy v1.6.0
goixy [flags]
  -host string
      host (default "127.0.0.1")
  -port string
      port (default "1080")
  -v
      verbose
  -vv
      very verbose
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
    "DirectPort": "12345"
}
```

You need to run lightsocks on `1.2.3.4:5678` and `127.0.0.1:12345`.

## how it works

![https://github.com/mitnk/goixy/blob/master/howitworks.png](https://github.com/mitnk/goixy/blob/master/howitworks.png)

## lightsocks

[https://github.com/mitnk/lightsocks](https://github.com/mitnk/lightsocks)
