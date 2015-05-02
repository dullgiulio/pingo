package pingo

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
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
	server  *rpc.Server
	objs    []string
	conf    *config
	running bool
}

func newRpcServer() *rpcServer {
	r := &rpcServer{
		server: rpc.DefaultServer,
		objs:   make([]string, 0),
		conf:   makeConfig(), // conf remains fixed after this point
	}
	r.register(&PingoRpc{})
	return r
}

var defaultServer = newRpcServer()

func (r *rpcServer) register(obj interface{}) {
	element := reflect.TypeOf(obj).Elem()
	r.objs = append(r.objs, element.Name())
	r.server.Register(obj)
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
		r.server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
		listener, err = net.Listen(r.conf.proto, r.conf.addr)
		if err == nil {
			break
		}
	}

	if err != nil {
		h.output("fatal", fmt.Sprintf("%s: Could not connect in %d attemps, using %s protocol", errorCodeConnFailed, conn.retries(), r.conf.proto))
		return err
	}

	h.output("ready", fmt.Sprintf("proto=%s addr=%s", r.conf.proto, r.conf.addr))
	if err := http.Serve(listener, nil); err != nil {
		h.output("fatal", fmt.Sprintf("err-http-serve: %s", err.Error()))
		return err
	}
	return nil
}
