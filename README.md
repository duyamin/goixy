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
    "Domains": [
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
[2017-06-18 14:58:36][0] goixy v1.7.1 without Direct Porxy
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
goixy [flags]
  -host string
        host (default "127.0.0.1")
  -port string
        port (default "1080")
  -s int
        time span to print reports in seconds (default 600)
  -t int
        time out on connections in seconds (default 3600)
  -v    verbose
  -vv
        very verbose
  -wd
        Use Direct proxy (for HTTP Porxy only)
  -wbl
        Use Black List (for HTTP Porxy only)
```

NOTE: currently `-wd`, `-wbl` only supports HTTP Proxy. Even set
`-withdirect`, accesses with Socks Porxy (i.e. `curl -x socks5://...`)
will always use `Host:Port` proxy.

## Blacklist Operations

```
$ ipy
import redis

r = redis.Redis(db=7)

r.hincrby('blacklist', 'adanalytics.com')
r.hincrby('whitelist', 'api.qq.com')

r.hdel('blacklist', 'baidu.com')
r.delete('oklist')
```

### Listing

```
$ py scripts/list-ok-list.py -h
$ py scripts/list-ok-list.py --list ok
```

## lightsocks

[https://github.com/mitnk/lightsocks](https://github.com/mitnk/lightsocks)
