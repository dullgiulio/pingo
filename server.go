// Copyright 2015 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package pingo

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// Register a new object this plugin exports. The object must be
// an exported symbol and obey all rules an object in the standard
// "rpc" module has to obey.
//
// Register will panic if called after Run.
func Register(obj interface{}) {
	if defaultServer.running {
		panic("Do not call Register after Run")
	}
	defaultServer.register(obj)
}

// Run will start all the necessary steps to make the plugin available.
func Run() error {
	if !flag.Parsed() {
		flag.Parse()
	}
	return defaultServer.run()
}

// Internal object for plugin control
type PingoRpc struct{}

// Default constructor for interal object. Do not call manually.
func NewPingoRpc() *PingoRpc {
	return &PingoRpc{}
}

// Internal RPC call to shut down a plugin. Do not call manually.
func (s *PingoRpc) Exit(status int, unused *int) error {
	os.Exit(status)
	return nil
}

type config struct {
	proto   string
	addr    string
	prefix  string
	unixdir string
}

func makeConfig() *config {
	c := &config{}
	flag.StringVar(&c.proto, "pingo:proto", "unix", "Protocol to use: unix or tcp")
	flag.StringVar(&c.unixdir, "pingo:unixdir", "", "Alternative directory for unix socket")
	flag.StringVar(&c.prefix, "pingo:prefix", "pingo", "Prefix to output lines")
	return c
}

type rpcServer struct {
	*rpc.Server
	secret  string
	objs    []string
	conf    *config
	running bool
}

func newRpcServer() *rpcServer {
	rand.Seed(time.Now().UTC().UnixNano())
	r := &rpcServer{
		Server: rpc.NewServer(),
		secret: randstr(64),
		objs:   make([]string, 0),
		conf:   makeConfig(), // conf remains fixed after this point
	}
	r.register(&PingoRpc{})
	return r
}

var defaultServer = newRpcServer()

type bufReadWriteCloser struct {
	*bufio.Reader
	r io.ReadWriteCloser
}

func newBufReadWriteCloser(r io.ReadWriteCloser) *bufReadWriteCloser {
	return &bufReadWriteCloser{Reader: bufio.NewReader(r), r: r}
}

func (b *bufReadWriteCloser) Write(data []byte) (int, error) {
	return b.r.Write(data)
}

func (b *bufReadWriteCloser) Close() error {
	return b.r.Close()
}

func readHeaders(brwc *bufReadWriteCloser) ([]byte, error) {
	var buf bytes.Buffer
	var headerEnd bool

	for {
		b, err := brwc.ReadByte()
		if err != nil {
			return []byte(""), err
		}

		buf.WriteByte(b)

		if b == '\n' {
			if headerEnd {
				break
			}
			headerEnd = true
		} else {
			headerEnd = false
		}
	}

	return buf.Bytes(), nil
}

func parseHeaders(brwc *bufReadWriteCloser, m map[string]string) error {
	headers, err := readHeaders(brwc)
	if err != nil {
		return err
	}

	r := bytes.NewReader(headers)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ": ", 2)
		if parts[0] == "" {
			continue
		}
		m[parts[0]] = parts[1]
	}

	return nil
}

func (r *rpcServer) authConn(token string) bool {
	if token != "" && token == r.secret {
		return true
	}
	return false
}

func (r *rpcServer) serveConn(conn io.ReadWriteCloser, h meta) {
	bconn := newBufReadWriteCloser(conn)
	defer bconn.Close()

	headers := make(map[string]string)
	if err := parseHeaders(bconn, headers); err != nil {
		h.output("error", err.Error())
		return
	}

	if r.authConn(headers["Auth-Token"]) {
		r.Server.ServeConn(bconn)
	}

	return
}

func (r *rpcServer) register(obj interface{}) {
	element := reflect.TypeOf(obj).Elem()
	r.objs = append(r.objs, element.Name())
	r.Server.Register(obj)
}

type connection interface {
	addr() string
	retries() int
}

type tcp int

func (t *tcp) addr() string {
	if *t < 1024 {
		// Only use unprivileged ports
		*t = 1023
	}

	*t = *t + 1
	return fmt.Sprintf("127.0.0.1:%d", *t)
}

func (t *tcp) retries() int {
	return 500
}

type unix string

func (u *unix) addr() string {
	name := randstr(8)
	if *u != "" {
		name = filepath.FromSlash(path.Join(string(*u), name))
	}
	return name
}

func (u *unix) retries() int {
	return 4
}

func (r *rpcServer) run() error {
	var conn connection
	var err error
	var listener net.Listener

	r.running = true

	h := meta(r.conf.prefix)
	h.output("objects", strings.Join(r.objs, ", "))

	switch r.conf.proto {
	case "tcp":
		conn = new(tcp)
	default:
		r.conf.proto = "unix"
		conn = new(unix)
	}

	for i := 0; i < conn.retries(); i++ {
		r.conf.addr = conn.addr()
		listener, err = net.Listen(r.conf.proto, r.conf.addr)
		if err == nil {
			break
		}
	}

	if err != nil {
		h.output("fatal", fmt.Sprintf("%s: Could not connect in %d attemps, using %s protocol", errorCodeConnFailed, conn.retries(), r.conf.proto))
		return err
	}

	h.output("auth-token", defaultServer.secret)
	h.output("ready", fmt.Sprintf("proto=%s addr=%s", r.conf.proto, r.conf.addr))
	for {
		var conn net.Conn
		conn, err = listener.Accept()
		if err != nil {
			h.output("fatal", fmt.Sprintf("err-http-serve: %s", err.Error()))
			continue
		}
		go r.serveConn(conn, h)
	}
}
