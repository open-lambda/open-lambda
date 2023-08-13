package websocket

import (
	"flag"
	"github.com/gobwas/ws"
	"github.com/urfave/cli/v2"
	"log"
	"net"
	"time"
)

var HttpPort string

var ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")

var manager *WsManager

type deadliner struct {
	net.Conn
	t time.Duration
}

type HandlerFunc func(interface{})

// wsHandler upgrade the http connection to websocket
func wsHandler(conn net.Conn) {
	safeConn := deadliner{conn, *ioTimeout}
	// Upgrade the connection to WebSocket
	_, err := ws.Upgrade(safeConn)

	if err != nil {
		log.Printf("%s: upgrade error: %v", nameConn(conn), err)
		conn.Close()
		return
	}
	log.Printf("established websocket connection: %s", nameConn(conn))
	client := &Client{conn: conn}
	client.onConnect()
}

// todo: integrate loadConf with the boss package
func loadConf() {
	// 5000 is the default http port of ol worker
	HttpPort = "5000"
	/*	var content []byte
		var err error
		for { // read the boss.json file
			content, err = ioutil.ReadFile("boss.json")
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		err = json.Unmarshal(content, &conf)
		if err != nil {
			log.Fatal(err)
		}*/
}

func Start(ctx *cli.Context) error {
	loadConf()
	host := ctx.String("host")
	port := ctx.String("port")
	url := host + ":" + port
	log.Println("ws-api listening on " + url)
	manager = NewWsManager()

	go manager.startInternalApi()

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
		}
		go wsHandler(conn)
	}
	return nil
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
