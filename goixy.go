package main

import (
	"crypto/sha256"
	"encoding/binary"
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
	"time"

	"github.com/mitnk/goutils/encrypt"
)

var VERSION = "1.2.0"
var KEY = getKey()
var countConnected = 0
var DEBUG = false
var VERBOSE = false

func main() {
	host := flag.String("host", "127.0.0.1", "host")
	port := flag.String("port", "1080", "port")
	rhost := flag.String("rhost", "", "remote host")
	rport := flag.String("rport", "", "remote port")
	debug := flag.Bool("v", false, "verbose")
	verbose := flag.Bool("vv", false, "very verbose")
	flag.Usage = func() {
		fmt.Printf("Usage of goixy v%s\n", VERSION)
		fmt.Printf("goixy [flags]\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()
	if *rhost == "" || *rport == "" {
		fmt.Printf("You need set rhost and rport\n")
		return
	}
	DEBUG = *debug
	VERBOSE = *verbose

	local, err := net.Listen("tcp", *host+":"+*port)
	if err != nil {
		fmt.Printf("net listen: %v\r", err)
		os.Exit(2)
	}
	defer local.Close()

	info("goixy v%s", VERSION)
	info("remote: %s:%s", *rhost, *rport)
	info("listen on port: %s:%s", *host, *port)

	for {
		client, err := local.Accept()
		if err != nil {
			continue
		}
		go handleClient(client, *rhost, *rport)
	}
}

func handleClient(client net.Conn, rhost, rport string) {
	countConnected += 1
	defer func() {
		countConnected -= 1
	}()
	defer client.Close()
	info("connected from %v.", client.RemoteAddr())

	data := make([]byte, 1)
	n, err := client.Read(data)
	if err != nil || n != 1 {
		info("cannot read init data from client")
		return
	}
	if data[0] == 5 {
		handleSocks(client, rhost, rport)
	} else if data[0] > 5 {
		handleHTTP(client, rhost, rport, data[0])
	} else {
		info("Error: only support HTTP and Socksv5")
	}
}

func handleSocks(client net.Conn, rhost, rport string) {
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
	info("server %s:%s", shost, sport)

	// reply to client to estanblish the socks v5 connection
	client.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	handleRemote(client, rhost, rport, shost, sport, nil, nil)
}

func handleHTTP(client net.Conn, rhost, rport string, firstByte byte) {
	dataInit := make([]byte, 8192)
	dataInit[0] = firstByte
	nDataInit, err := client.Read(dataInit[1:])
	nDataInit = nDataInit + 1 // plus firstByte
	if err != nil {
		info("cannot read init data from client.")
		return
	}
	isForHTTPS := strings.HasPrefix(string(dataInit[:nDataInit]), "CONNECT")

	endor := " HTTP/"
	re := regexp.MustCompile(" .*" + endor)
	s := re.FindString(string(dataInit[:nDataInit]))
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
	info("server %s:%s", shost, sport)

	var d2c []byte
	var d2r []byte
	if isForHTTPS {
		d2c = []byte("HTTP/1.0 200 OK\r\n\r\n")
	} else {
		dataInit := encrypt.Encrypt(dataInit[:nDataInit], KEY)
		dataInitLen := make([]byte, 2)
		binary.BigEndian.PutUint16(dataInitLen, uint16(len(dataInit)))
		d2r = append(dataInitLen, dataInit...)
	}
	handleRemote(client, rhost, rport, shost, sport, d2c, d2r)
}

func handleRemote(client net.Conn, rhost, rport, shost, sport string, d2c, d2r []byte) {
	remote, err := net.Dial("tcp", rhost+":"+rport)
	if err != nil {
		info("cannot connect to remote: %s:%s", rhost, rport)
		return
	}
	defer remote.Close()
	info("connected to remote: %s", remote.RemoteAddr())

	bytesCheck := make([]byte, 8)
	copy(bytesCheck, KEY[8:16])
	bytesCheck = encrypt.Encrypt(bytesCheck, KEY)
	remote.Write([]byte{byte(len(bytesCheck))})
	remote.Write(bytesCheck)

	bytesHost := []byte(shost)
	bytesHost = encrypt.Encrypt(bytesHost, KEY)
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
	go readDataFromRemote(ch_remote, remote)

	shouldStop := false
	for {
		if shouldStop {
			break
		}

		select {
		case data := <-ch_remote:
			if data == nil {
				remote.Close()
				debug("remote closed")
				shouldStop = true
				break
			}
			client.Write(data)
		case di := <-ch_client:
			if di.data == nil {
				client.Close()
				debug("client closed")
				shouldStop = true
				break
			}
			buffer := encrypt.Encrypt(di.data[:di.size], KEY)
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(len(buffer)))
			remote.Write(b)
			remote.Write(buffer)
			debug("sent %d bytes to remote", len(buffer))
		}
	}
}

func readDataFromClient(ch chan DataInfo, ch2 chan []byte, conn net.Conn) {
	for {
		data := make([]byte, 8192)
		n, err := conn.Read(data)
		if err != nil {
			ch <- DataInfo{nil, 0}
			ch2 <- nil
			return
		}
		debug("received %d bytes from client", n)
		verbose("client: %s", data[:n])
		ch <- DataInfo{data, n}
	}
}

func readDataFromRemote(ch chan []byte, conn net.Conn) {
	for {
		buffer := make([]byte, 2)
		_, err := io.ReadFull(conn, buffer)
		if err != nil {
			ch <- nil
			return
		}
		size := binary.BigEndian.Uint16(buffer)
		buffer = make([]byte, size)
		_, err = io.ReadFull(conn, buffer)
		if err != nil {
			ch <- nil
			return
		}
		data, err := encrypt.Decrypt(buffer, KEY)
		if err != nil {
			info("ERROR: cannot decrypt data from client")
			ch <- nil
			return
		}
		debug("received %d bytes from remote", len(data))
		verbose("remote: %s", data)
		ch <- data
	}
}

func getKey() []byte {
	usr, err := user.Current()
	if err != nil {
		fmt.Printf("user current: %v\r", err)
		os.Exit(2)
	}
	fileKey := path.Join(usr.HomeDir, ".lightsockskey")
	data, err := ioutil.ReadFile(fileKey)
	if err != nil {
		fmt.Printf("failed to load key file: %v\r", err)
		os.Exit(1)
	}
	s := strings.TrimSpace(string(data))
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func info(format string, a ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s][%d] ", ts, countConnected)
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

type DataInfo struct {
	data []byte
	size int
}

const ATYP_IPV4 = 1
const ATYP_DOMAIN = 3
const ATYP_IPV6 = 4
