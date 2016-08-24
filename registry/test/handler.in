package balancer

import (
	"bytes"
	"container/list"
	"io"
	"net"

	"github.com/open-lambda/load-balancer/balancer/connPeek"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
	"google.golang.org/grpc/transport"
)

type LoadBalancer struct {
	Chooser   serverPick.ServerPicker
	Address   string
	Consumers int
	Conns     chan *net.TCPConn
}

// Actually send the request to "best" backend (first one for now)
func (lb *LoadBalancer) ForwardRequest(peekconn *connPeek.ReaderConn, clientconn, serverconn *net.TCPConn, buf bytes.Buffer) {
	peekconn.SecondConn(serverconn) // This is so we can close serverconn cleanly
	io.Copy(serverconn, &buf)
	io.Copy(clientconn, serverconn)
}

func (lb *LoadBalancer) HandleConn(clientconn *net.TCPConn) {
	// Will need access to buf later for proxying
	var buf bytes.Buffer
	r := io.TeeReader(clientconn, &buf)

	// Using conn to peek at the method name w/o affecting buf
	peekconn := &connPeek.ReaderConn{Reader: r, Conn: clientconn}
	st, err := transport.NewServerTransport("http2", peekconn, 100, nil)
	if err != nil {
		panic(err)
	}

	st.HandleStreams(func(stream *transport.Stream) {
		// Get method name
		name := stream.Method()

		// Make decision about which backend(s) to connect to
		servers, err := lb.Chooser.ChooseServers(name, *list.New())
		if err != nil {
			panic(err)
		}

		serveraddr, err := net.ResolveTCPAddr("tcp", servers[0])
		if err != nil {
			panic(err)
		}

		// nil argument is the local address
		serverconn, err := net.DialTCP("tcp", nil, serveraddr)
		if err != nil {
			panic(err)
		}

		go lb.ForwardRequest(peekconn, clientconn, serverconn, buf)
	})
}

func (lb *LoadBalancer) ConnConsumer() {
	for {
		conn := <-lb.Conns
		lb.HandleConn(conn)
	}
}

func (lb *LoadBalancer) Run() {
	tcpaddr, err := net.ResolveTCPAddr("tcp", lb.Address)
	if err != nil {
		panic(err)
	}

	lis, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		panic(err)
	}

	for i := 0; i < lb.Consumers; i++ {
		go lb.ConnConsumer()
	}

	for {
		conn, err := lis.AcceptTCP()
		if err != nil {
			panic(err)
		}
		lb.Conns <- conn
	}
}

// TODO add support for multiple addresses?
func (lb *LoadBalancer) Init(address string, chooser serverPick.ServerPicker, consumers int) {
	lb.Address = address
	lb.Chooser = chooser
	lb.Consumers = consumers
	lb.Conns = make(chan *net.TCPConn)
}
