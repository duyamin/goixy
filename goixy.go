package main

import (
	"fmt"
	"flag"
	"net"
	"io"
	"io/ioutil"
	"encoding/binary"
	"crypto/sha256"
	"os/user"
	"path"
	"strconv"
	"strings"
	"net/url"
	"regexp"
	"time"

	"github.com/mitnk/goutils/encrypt"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host")
	port := flag.String("port", "11080", "port")
	rhost := flag.String("rhost", "", "remote host")
	rport := flag.String("rport", "", "remote port")
	flag.Usage = func() {
        fmt.Printf("lightsocks [flags]\nwhere flags are:\n")
        flag.PrintDefaults()
    }
    flag.Parse()
	runLocal(*host, *port, *rhost, *rport)
}

var countConnected = 0
type DataInfo struct {
	data []byte
	size int
}

func runLocal(host, port, rhost, rport string) {
	local, err := net.Listen("tcp", host + ":" + port)
	check(err)
	defer local.Close()
	info("remote: %s:%s", rhost, rport)
	info("listen on port: %s:%s", host, port)

	for {
        client, err := local.Accept()
        if err != nil {
            continue
        }
        go handleClient(client, rhost, rport)
    }
}

func handleClient(client net.Conn, rhost, rport string) {
	countConnected += 1
	defer func() {
		countConnected -= 1
	}()
    defer client.Close()
	info("connected from %v.", client.RemoteAddr())

	dataInit := make([]byte, 8192)
	nDataInit, err := client.Read(dataInit)
	if err != nil {
		fmt.Printf("cannot read init data from client.\n")
		return
	}
	isForHTTPS := strings.HasPrefix(string(dataInit[:nDataInit]), "CONNECT")

	endor := " HTTP/"
	re := regexp.MustCompile(" .*" + endor)
	s := re.FindString(string(dataInit[:nDataInit]))
	s = s[1:len(s) - len(endor)]
	info("url: %s", s)
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "http://" + s
		info("+ url: %s", s)
	}
	u, err := url.Parse(s)
	if err != nil {
		fmt.Printf("bad url: %s", s)
		return
	}
	portServer := ""
	hostServer := ""
	host_, port_, _ := net.SplitHostPort(u.Host)
	if port_ != "" {
		portServer = port_
		hostServer = host_
	} else {
		portServer = "80"
		hostServer = u.Host
	}
	info("host: %s", hostServer)
	info("port: %s", portServer)

	remote, err := net.Dial("tcp", rhost + ":" + rport)
	if err != nil {
		fmt.Printf("cannot connect to remote: %s:%s\n", rhost, rport)
		return
	}
	defer remote.Close()
	info("connected to remote: %s", remote.RemoteAddr())

	key := getKey()
	bytesHost := []byte(hostServer)
	bytesHost = encrypt.Encrypt(bytesHost, key[:])
	remote.Write([]byte{byte(len(bytesHost))})
	remote.Write(bytesHost)

	b := make([]byte, 2)
	nportServer, _ := strconv.Atoi(portServer)
	binary.BigEndian.PutUint16(b, uint16(nportServer))
	remote.Write(b)

	ch_client := make(chan DataInfo)
	ch_remote := make(chan []byte)

	if isForHTTPS {
		client.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	} else {
		dataInit := encrypt.Encrypt(dataInit[:nDataInit], key[:])
		dataInitLen := make([]byte, 2)
		binary.BigEndian.PutUint16(dataInitLen, uint16(len(dataInit)))
		remote.Write(dataInitLen)
		remote.Write(dataInit)
	}

	go readDataFromClient(ch_client, ch_remote, client)
	go readDataFromRemote(ch_remote, remote, key[:])

	shouldStop := false
	for {
		if shouldStop {
			break
		}

		select {
		case data := <-ch_remote:
			if data == nil {
				remote.Close()
				info("remote closed.")
				break
			}
			client.Write(data)
		case di := <-ch_client:
			if di.data == nil {
				client.Close()
				info("client closed.")
				shouldStop = true
				break
			}
			info("received %d bytes from client", di.size)
			buffer := encrypt.Encrypt(di.data[:di.size], key[:])
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(len(buffer)))
			remote.Write(b)
			remote.Write(buffer)
			info("sent %d bytes to remote", len(buffer))
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
		info("received %d bytes from client", n)
		ch <- DataInfo{data, n}
	}
}

func readDataFromRemote(ch chan []byte, conn net.Conn, key []byte) {
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
		data, err := encrypt.Decrypt(buffer, key)
		if err != nil {
			fmt.Printf("ERROR: cannot decrypt data from client.")
			ch <- nil
			return
		}
		info("received %d bytes from remote", len(data))
		ch <- data
	}
}

func getKey() [32]byte {
	usr, err := user.Current()
	check(err)
	fileKey := path.Join(usr.HomeDir, ".lightsockskey")
	data, err := ioutil.ReadFile(fileKey)
	s := strings.TrimSpace(string(data))
	check(err)
	return sha256.Sum256([]byte(s))
}

func check(err error) {
	if (err != nil) {
		panic(err)
	}
}

func info(format string, a...interface{}) (n int, err error) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s][%d] ", ts, countConnected)
	return fmt.Printf(prefix + format + "\n", a...)
}
