package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

type Client struct {
	conn net.Conn
	fd   int
	id   uuid.UUID

	wsPacket WsPacket
	writeMux sync.Mutex
	event    *Event
}

type WsPacket struct {
	Action string                 `json:"action"` // route to the corresponding handler, for now it's only `run`
	Target string                 `json:"target"`
	Body   map[string]interface{} `json:"body"`
}

type Event struct {
	Context *Context                `json:"context"`
	Body    *map[string]interface{} `json:"body"`
}

type Context struct {
	Id uuid.UUID `json:"id"`
}

// onConnect is called when a new client is connected.
// it registers the client to the manager and sends the client its ID
func (client *Client) onConnect() {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	client.id = id
	manager.RegisterClient(client)

	// call onConnect func in lambda
	client.event = &Event{&Context{Id: client.id}, nil}
	client.wsPacket.Target = "onConnect"
	client.run()
}

func (client *Client) onDisconnect() {
	manager.UnregisterClient(client)
	err := client.conn.Close()
	if err != nil {
		return
	}

	// call onDisconnect func in lambda
	client.event = &Event{&Context{Id: client.id}, nil}
	client.wsPacket.Target = "onDisconnect"
	client.run()
}

func (client *Client) run() {
	respBody, _, err := client.sendRequest()
	if err != nil {
		log.Println(err)
		log.Println(respBody)
	}
	client.write(respBody)
}

func (client *Client) write(body []byte) {
	client.writeMux.Lock()
	wsutil.WriteServerText(client.conn, body)
	client.writeMux.Unlock()
}

var reqCnt int32 = 0

// sendRequest sends an HTTP request to local ol worker and returns the response body
func (client *Client) sendRequest() ([]byte, int, error) {
	atomic.AddInt32(&reqCnt, 1)
	println("reqeust Cnt:", reqCnt)

	url := "http://localhost:" + HttpPort + "/run/" + client.wsPacket.Target

	eventBytes, _ := json.Marshal(client.event)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(eventBytes))
	if err != nil {
		return nil, -1, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	var respBuf bytes.Buffer
	_, err = io.Copy(&respBuf, resp.Body)
	if err != nil {
		return nil, -1, err
	}
	return respBuf.Bytes(), resp.StatusCode, nil
}
