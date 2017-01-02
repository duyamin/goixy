package main

import (
	"fmt"
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

	"github.com/mitnk/goutils/encrypt"
)

type DataInfo struct {
	data []byte
	size int
}

var localForDebug = false

func runLocal(host, port, rhost, rport string, debug bool) {
	localForDebug = debug
	server, err := net.Listen("tcp", host + ":" + port)
	check(err)
	defer server.Close()
	fmt.Printf("listen on port %s:%s\n", host, port)

	for {
        client, err := server.Accept()
        if err != nil {
            continue
        }
        go handleClient(client, rhost, rport)
    }
}

func handleClient(client net.Conn, rhost, rport string) {
    defer client.Close()
	info("connected from %v.", client.RemoteAddr())

	dataInit := make([]byte, 8192)
	nDataInit, err := client.Read(dataInit)
	if err != nil {
		fmt.Printf("cannot read init data from client.\n")
		return
	}
	info("dataInit: %s", dataInit[:nDataInit])

	isForHTTPS := strings.HasPrefix(string(dataInit[:nDataInit]), "CONNECT")
	info("isForHTTPS: %v", isForHTTPS)

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
	check(err)
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
	info("connected to remote server: %s", remote.RemoteAddr())

	key := getKey()
	bytesHost := []byte(hostServer)
	bytesHost = encrypt.Encrypt(bytesHost, key[:])
	info("len bytesHost: %d", len(bytesHost))
	remote.Write([]byte{byte(len(bytesHost))})
	nsent, _ := remote.Write(bytesHost)
	info("bytes of host sent. [%d/%d]", len(bytesHost), nsent)

	b := make([]byte, 2)
	nportServer, _ := strconv.Atoi(portServer)
	binary.BigEndian.PutUint16(b, uint16(nportServer))
	nsent, _ = remote.Write(b)
	info("bytes of port sent. [%d/%d]", len(b), nsent)

	ch_client := make(chan DataInfo)
	ch_remote := make(chan []byte)

	foo1 := encrypt.Encrypt(dataInit[:nDataInit], key[:])
	foo2 := make([]byte, 2)
	binary.BigEndian.PutUint16(foo2, uint16(len(foo1)))
	remote.Write(foo2)
	remote.Write(foo1)

	go readDataFromClient(ch_client, client)
	go readDataFromRemote(ch_remote, remote, key[:])

	for {
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

func readDataFromClient(ch chan DataInfo, conn net.Conn) {
	for {
		data := make([]byte, 8192)
		n, err := conn.Read(data)
		if err != nil {
			ch <- DataInfo{nil, 0}
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
	if !localForDebug {
		return 0, nil
	}
	return fmt.Printf(format + "\n", a...)
}