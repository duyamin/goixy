package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/mitnk/goutils/encrypt"
	"github.com/orcaman/concurrent-map"
)

type GoixyConfig struct {
	Host       string
	Port       string
	Key        string
	DomainList  []string
	DirectHost string
	DirectPort string
	DirectKey  string
}

var GC GoixyConfig = GoixyConfig{}

var VERSION = "1.8.0"
var KEY = []byte("")
var DIRECT_KEY = []byte("")
var COUNT_CONNECTED = 0
var DEBUG = false
var VERBOSE = false
var WITH_DIRECT = false
var WITH_BLACK_LIST = false
var SPAN_TIMEOUT int64 = 3600

var SERVER_INFO = cmap.New()
var MUTEX = &sync.Mutex{}
var REDIS_CLI *redis.Client = nil
var REDIS_DB = 7

func main() {
	host := flag.String("host", "127.0.0.1", "host")
	port := flag.String("port", "1080", "port")
	with_black_list := flag.Bool("wbl", false, "Use balcklist (for HTTP only)")
	with_direct := flag.Bool("wd", false, "Use Direct proxy (for HTTP Porxy only)")
	redis_db := flag.Int("db", 7, "Redis DB index")
	_debug := flag.Bool("v", false, "verbose, print some debug info")
	verbose := flag.Bool("vv", false, "very verbose, more debug info")
	_span_timeout := flag.Int64("t", 3600, "time out on connections in seconds")
	flag.Usage = func() {
		fmt.Printf("Usage of goixy v%s\n", VERSION)
		fmt.Printf("goixy [FLAGS]\n\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()
	DEBUG = *_debug
	SPAN_TIMEOUT = *_span_timeout
	if SPAN_TIMEOUT < 60 {
		SPAN_TIMEOUT = 60
	}
	VERBOSE = *verbose
	WITH_BLACK_LIST = *with_black_list
	WITH_DIRECT = *with_direct
	REDIS_DB = *redis_db
	// REDIS_CLI: let me be last call
	REDIS_CLI = get_redis_client()
	loadRouterConfig()

	local, err := net.Listen("tcp", *host+":"+*port)
	if err != nil {
		fmt.Printf("net listen: %v\r", err)
		os.Exit(2)
	}
	defer local.Close()

	_with_or_not := "with"
	if !WITH_DIRECT {
		_with_or_not = "without"
	}
	info("goixy v%s %s Direct Porxy", VERSION, _with_or_not)
	info("listen on port: %s:%s", *host, *port)

	for {
		client, err := local.Accept()
		if err != nil {
			continue
		}
		go handleClient(client)
	}
}

func handleClient(client net.Conn) {
	MUTEX.Lock()
	COUNT_CONNECTED += 1
	MUTEX.Unlock()
	defer func() {
		client.Close()
		MUTEX.Lock()
		COUNT_CONNECTED -= 1
		MUTEX.Unlock()
		debug("closed client")
	}()
	debug("connected from %v.", client.RemoteAddr())

	data := make([]byte, 1)
	n, err := client.Read(data)
	if err != nil || n != 1 {
		info("cannot read init data from client")
		return
	}
	if data[0] == 5 {
		verbose("handle with socks v5")
		handleSocks(client)
	} else if data[0] > 5 {
		verbose("handle with http")
		handleHTTP(client, data[0])
	} else {
		info("Error: only support HTTP and Socksv5")
	}
}

func handleSocks(client net.Conn) {
	buffer := make([]byte, 1)
	_, err := io.ReadFull(client, buffer)
	if err != nil {
		info("cannot read from client")
		return
	}
	buffer = make([]byte, buffer[0])
	_, err = io.ReadFull(client, buffer)
	if err != nil {
		info("cannot read from client")
		return
	}
	if !byteInArray(0, buffer) {
		info("client not support bare connect")
		return
	}

	// send initial SOCKS5 response (VER, METHOD)
	client.Write([]byte{5, 0})

	buffer = make([]byte, 4)
	_, err = io.ReadFull(client, buffer)
	if err != nil {
		info("failed to read (ver, cmd, rsv, atyp) from client")
		return
	}
	ver, cmd, atyp := buffer[0], buffer[1], buffer[3]
	if ver != 5 {
		info("ver should be 5, got %v", ver)
		return
	}
	// 1: connect 2: bind
	if cmd != 1 && cmd != 2 {
		info("bad cmd:%v", cmd)
		return
	}
	shost := ""
	sport := ""
	if atyp == ATYP_IPV6 {
		info("do not support ipv6 yet")
		return
	} else if atyp == ATYP_DOMAIN {
		buffer = make([]byte, 1)
		_, err = io.ReadFull(client, buffer)
		if err != nil {
			info("cannot read from client")
			return
		}
		buffer = make([]byte, buffer[0])
		_, err = io.ReadFull(client, buffer)
		if err != nil {
			info("cannot read from client")
			return
		}
		shost = string(buffer)
	} else if atyp == ATYP_IPV4 {
		buffer = make([]byte, 4)
		_, err = io.ReadFull(client, buffer)
		if err != nil {
			info("cannot read from client")
			return
		}
		shost = net.IP(buffer).String()
	} else {
		info("bad atyp: %v", atyp)
		return
	}

	buffer = make([]byte, 2)
	_, err = io.ReadFull(client, buffer)
	if err != nil {
		info("cannot read port from client")
		return
	}
	sport = fmt.Sprintf("%d", binary.BigEndian.Uint16(buffer))
	info("socks target: %s:%s", shost, sport)

	// reply to client to estanblish the socks v5 connection
	client.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	rhost, rport, key := getRemoteInfo(shost, true)
	handleRemote(client, shost, sport, rhost, rport, nil, nil, key)
}

func handleHTTP(client net.Conn, firstByte byte) {
	dataInit := make([]byte, 8192)
	dataInit[0] = firstByte
	nDataInit, err := client.Read(dataInit[1:])
	nDataInit = nDataInit + 1 // plus firstByte
	if err != nil {
		info("cannot read init data from client.")
		return
	}
	isForHTTPS := strings.HasPrefix(string(dataInit[:nDataInit]), "CONNECT")
	verbose("isForHTTPS: %v", isForHTTPS)
	verbose("got content from client:\n%s", dataInit[:nDataInit])

	endor := " HTTP/"
	re := regexp.MustCompile(" .*" + endor)
	s := re.FindString(string(dataInit[:nDataInit]))
	if s == "" {
		// no url found. not valid http proxy protocol?
		return
	}

	s = s[1 : len(s)-len(endor)]
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		info("bad url: %s", s)
		return
	}
	sport := ""
	shost := ""
	host_, port_, _ := net.SplitHostPort(u.Host)
	if port_ != "" {
		sport = port_
		shost = host_
	} else {
		sport = "80"
		shost = u.Host
	}
	if is_in_item_list("blacklist", shost) {
		info("closed black site: %s", shost)
		return
	}

	inc_item_count("oklist", shost)

	var http_type string
	if isForHTTPS {
		http_type = "HTTPS"
	} else {
		http_type = "http"
	}
	info("[%s] target: %s:%s", http_type, shost, sport)

	rhost, rport, key := getRemoteInfo(shost, false)
	var d2c []byte
	var d2r []byte
	if isForHTTPS {
		d2c = []byte("HTTP/1.0 200 OK\r\n\r\n")
	} else {
		// dataInit := encrypt.Encrypt(dataInit[:nDataInit], key)
		reg1, _ := regexp.Compile("^HEAD https?:..[^/]+/")
		path := reg1.ReplaceAllString(string(dataInit[:nDataInit]), "HEAD /")
		reg2, _ := regexp.Compile("^GET https?:..[^/]+/")
		path = reg2.ReplaceAllString(string(path), "GET /")
		dataInit := encrypt.Encrypt([]byte(path), key)
		dataInitLen := make([]byte, 2)
		binary.BigEndian.PutUint16(dataInitLen, uint16(len(dataInit)))
		d2r = append(dataInitLen, dataInit...)
	}
	handleRemote(client, shost, sport, rhost, rport, d2c, d2r, key)
}

func getRemoteInfo(shost string, is_socks bool) (string, string, []byte) {
	rhost := ""
	rport := ""
	key := []byte("")
	if !is_socks && WITH_DIRECT && !is_in_domain_list(shost) {
		rhost = GC.DirectHost
		rport = GC.DirectPort
		key = DIRECT_KEY
	} else {
		rhost = GC.Host
		rport = GC.Port
		key = KEY
	}
	return rhost, rport, key
}

func handleRemote(client net.Conn, shost, sport, rhost, rport string, d2c, d2r, key []byte) {
	remote, err := net.Dial("tcp", rhost+":"+rport)
	if err != nil {
		info("cannot connect to remote: %s:%s", rhost, rport)
		return
	}
	keyServer := fmt.Sprintf("%s:%s", shost, sport)
	initServers(keyServer, 0)
	defer func() {
		remote.Close()
		deleteServers(fmt.Sprintf("%s:%s", shost, sport))
		debug("closed remote for %s:%s", shost, sport)
	}()
	debug("connected to remote: %s", remote.RemoteAddr())

	bytesCheck := make([]byte, 8)
	copy(bytesCheck, key[8:16])
	bytesCheck = encrypt.Encrypt(bytesCheck, key)
	remote.Write([]byte{byte(len(bytesCheck))})
	remote.Write(bytesCheck)

	bytesHost := []byte(shost)
	bytesHost = encrypt.Encrypt(bytesHost, key)
	remote.Write([]byte{byte(len(bytesHost))})
	remote.Write(bytesHost)

	b := make([]byte, 2)
	nportServer, _ := strconv.Atoi(sport)
	binary.BigEndian.PutUint16(b, uint16(nportServer))
	remote.Write(b)

	ch_client := make(chan DataInfo)
	ch_remote := make(chan []byte)

	if d2c != nil {
		client.Write(d2c)
	}
	if d2r != nil {
		remote.Write(d2r)
	}

	go readDataFromClient(ch_client, ch_remote, client)
	go readDataFromRemote(ch_remote, remote, shost, sport, key)

	for {
		select {
		case data, ok := <-ch_remote:
			if !ok {
				return
			}
			client.Write(data)
		case di, ok := <-ch_client:
			if !ok {
				return
			}
			buffer := encrypt.Encrypt(di.data[:di.size], key)
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(len(buffer)))
			remote.Write(b)
			remote.Write(buffer)
		case <-time.After(time.Second * time.Duration(SPAN_TIMEOUT)):
			debug("timeout on %s:%s", shost, sport)
			return
		}
	}
}

func readDataFromClient(ch chan DataInfo, ch2 chan []byte, conn net.Conn) {
	for {
		data := make([]byte, 8192)
		n, err := conn.Read(data)
		if err != nil {
			close(ch)
			break
		}
		debug("received %d bytes from client", n)
		verbose("client: %s", data[:n])
		ch <- DataInfo{data, n}
	}
}

func readDataFromRemote(ch chan []byte, conn net.Conn, shost, sport string, key []byte) {
	for {
		buffer := make([]byte, 2)
		_, err := io.ReadFull(conn, buffer)
		if err != nil {
			break
		}
		size := binary.BigEndian.Uint16(buffer)

		keyServer := fmt.Sprintf("%s:%s", shost, sport)
		inc_item_count_by("byteslist", keyServer, int64(size))

		buffer = make([]byte, size)
		_, err = io.ReadFull(conn, buffer)
		if err != nil {
			break
		}
		data, err := encrypt.Decrypt(buffer, key)
		if err != nil {
			info("ERROR: cannot decrypt data from client")
			break
		}
		n_bytes := len(data)
		debug("[%s:%s] received %d bytes", shost, sport, n_bytes)
		verbose("remote: %s", data)
		ch <- data
	}
	close(ch)
}

func loadDirects() []byte {
	usr, err := user.Current()
	if err != nil {
		fmt.Printf("user current: %v\n", err)
		os.Exit(2)
	}
	fileKey := path.Join(usr.HomeDir, ".lightsockskey")
	data, err := ioutil.ReadFile(fileKey)
	if err != nil {
		fmt.Printf("failed to load key file: %v\n", err)
		os.Exit(1)
	}
	s := strings.TrimSpace(string(data))
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func getRouterConfig() []byte {
	usr, err := user.Current()
	if err != nil {
		fmt.Printf("user current: %v\n", err)
		os.Exit(2)
	}
	fileConfig := path.Join(usr.HomeDir, ".goixy/config.json")
	if _, err := os.Stat(fileConfig); os.IsNotExist(err) {
		fmt.Printf("config file is missing: %v\n", fileConfig)
		os.Exit(2)
	}

	data, err := ioutil.ReadFile(fileConfig)
	if err != nil {
		fmt.Printf("failed to load direct-servers file: %v\n", err)
		os.Exit(1)
	}
	return data
}

func info(format string, a ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s][%d] ", ts, COUNT_CONNECTED)
	fmt.Printf(prefix+format+"\n", a...)
}

func debug(format string, a ...interface{}) {
	if DEBUG || VERBOSE {
		info(format, a...)
	}
}

func verbose(format string, a ...interface{}) {
	if VERBOSE {
		info(format, a...)
	}
}

func byteInArray(b byte, A []byte) bool {
	for _, e := range A {
		if e == b {
			return true
		}
	}
	return false
}

func initServers(key string, bytes int64) {
	MUTEX.Lock()
	defer MUTEX.Unlock()

	if m, ok := SERVER_INFO.Get(key); ok {
		if tmp, ok := m.(cmap.ConcurrentMap).Get("count"); ok {
			m.(cmap.ConcurrentMap).Set("count", tmp.(int64) + 1)
		}
	} else {
		m := cmap.New()
		now := time.Now()
		m.Set("count", int64(1))
		m.Set("bytes", bytes)
		m.Set("ts", now.Unix())
		SERVER_INFO.Set(key, m)
	}
}

func deleteServers(key string) {
	MUTEX.Lock()
	defer MUTEX.Unlock()

	if m, ok := SERVER_INFO.Get(key); ok {
		if tmp, ok := m.(cmap.ConcurrentMap).Get("count"); ok {
			count := tmp.(int64)
			if count <= 1 {
				SERVER_INFO.Remove(key)
			} else {
				m.(cmap.ConcurrentMap).Set("count", count - 1)
			}
		}
	}
}

func loadRouterConfig() {
	b := getRouterConfig()
	if b == nil {
		return
	}
	err := json.Unmarshal(b, &GC)
	if err != nil {
		fmt.Printf("Invalid Goixy Config: %v\n", err)
		os.Exit(2)
	}

	// init keys
	s := strings.TrimSpace(GC.Key)
	_tmp := sha256.Sum256([]byte(s))
	KEY = _tmp[:]
	if GC.DirectKey != "" {
		s = strings.TrimSpace(GC.DirectKey)
		_tmp = sha256.Sum256([]byte(s))
		DIRECT_KEY = _tmp[:]
	} else {
		DIRECT_KEY = KEY
	}
}

func get_redis_client() *redis.Client {
	if !WITH_BLACK_LIST {
		return nil
	}
	cli := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "", // no password set
		DB:       REDIS_DB,
	})
	return cli
}

func inc_item_count(list_name, key string) {
	if REDIS_CLI == nil {
		return
	}
	REDIS_CLI.HIncrBy(list_name, key, 1)
}

func inc_item_count_by(list_name, key string, val int64) {
	if REDIS_CLI == nil {
		return
	}
	REDIS_CLI.HIncrBy(list_name, key, val)
}

func is_in_item_list(list_name string, shost string) bool {
	if !WITH_BLACK_LIST || REDIS_CLI == nil {
		return false
	}
	if list_name == "blacklist" && is_in_item_list("whitelist", shost) {
		return false
	}

	item_list := get_item_list(list_name)
	for _, ptn := range item_list {
		re := regexp.MustCompile(ptn)
		if re.FindString(shost) != "" {
			inc_item_count(list_name, ptn)
			return true
		}
	}
	return false
}

func get_item_list(list_name string) []string {
	if !WITH_BLACK_LIST || REDIS_CLI == nil {
		return nil
	}
	result, _ := REDIS_CLI.HKeys(list_name).Result()
	return result
}

func is_in_domain_list(shost string) bool {
	for _, s := range GC.DomainList {
		re := regexp.MustCompile(s)
		s := re.FindString(shost)
		if s != "" {
			return true
		}
	}
	if is_in_item_list("domainlist", shost) {
		return true
	}
	return false
}

func fmtTimeSpan(n_seconds int64) string {
	str_span := ""
	if n_seconds > 3600 * 24 {
		str_span += fmt.Sprintf("%dd", n_seconds/(3600*24))
	}
	if n_seconds > 3600 {
		str_span += fmt.Sprintf("%dh", (n_seconds % (3600*24))/3600)
	}
	if n_seconds > 60 {
		str_span += fmt.Sprintf("%dm", (n_seconds%3600)/60)
	}
	str_span += fmt.Sprintf("%ds", n_seconds % 60)
	return str_span
}

type DataInfo struct {
	data []byte
	size int
}

const ATYP_IPV4 = 1
const ATYP_DOMAIN = 3
const ATYP_IPV6 = 4
