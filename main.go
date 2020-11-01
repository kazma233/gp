package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

// URLInfo uriInfo
type URLInfo struct {
	Method    string
	Host      string
	ReadBytes []byte
}

func main() {
	port := *flag.String("p", "7890", "listening port")
	flag.Parse()

	address := ":" + port
	fmt.Println("prepare listening: " + address)
	lis, err := net.Listen("tcp", address)
	checkErr(err)

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Printf("can`t accept: %v\n", err.Error())
			continue
		}

		log.Printf("one %s", conn.RemoteAddr().String())

		conn.SetDeadline(time.Now().Add(30 * time.Second))
		go handleAccept(conn)
	}
}

func handleAccept(conn net.Conn) {
	urlInfo, err := GetMethodAndHost(conn)
	if err != nil {
		log.Printf("get url info error: %v\n", err)
		return
	}

	address, err := GetRealAddress(urlInfo.Method, urlInfo.Host)
	if err != nil {
		log.Printf("get real url faild: %v\n", err)
		return
	}

	// proxy request
	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("dial faild: %v\n", err)
		return
	}

	if urlInfo.Method == "CONNECT" { // it`s
		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		server.Write(urlInfo.ReadBytes)
	}

	go io.Copy(server, conn)
	io.Copy(conn, server)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func GetMethodAndHost(conn net.Conn) (*URLInfo, error) {
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return nil, err
	}

	var method, host string
	linePosition := bytes.IndexByte(b, '\n')
	requestLine := string(b[:linePosition]) // GET http://xxx.com/ HTTP/1.1
	fmt.Sscanf(requestLine, "%s%s", &method, &host)

	return &URLInfo{method, host, b[:n]}, nil
}

func GetRealAddress(method, host string) (string, error) {
	forwardURL, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	var address string
	if forwardURL.Opaque == "443" { // https
		address = forwardURL.Scheme + ":443"
	} else { // http
		address = forwardURL.Host
		if strings.Index(forwardURL.Host, ":") == -1 {
			address += ":80"
		}
	}
	return address, nil
}
