package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
)

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

type client struct {
	conn net.Conn
	id   string
}

type Reply struct {
	SenderId string `json:"sender_id"`
	Body     string `json:"body"`
}

// deploy websocket_chat to your ol instance and
// start ol worker and websocket server before running the tests
var ip = "172.30.131.147"
var WsServerAddr = "ws://" + ip + ":4999"

func TestOnConnect(t *testing.T) {
	n := 100
	rdb := redis.NewClient(&redis.Options{
		Addr:     "172.30.131.147:6379",
		Password: "",
		DB:       0,
	})
	ctx := context.Background()

	num, err := rdb.SCard(ctx, "clients").Result()
	if err != nil {
		panic(err)
	}
	if num != 0 {
		t.Fatalf("Expected 0 clients before starting test")
	}
	connections := makeWsConnections(t, n)

	defer func() {
		err := rdb.FlushAll(ctx).Err()
		if err != nil {
			panic(err)
		}
		for i := 0; i < n; i++ {
			connections[i].conn.Close()
		}
	}()

	num, err = rdb.SCard(ctx, "clients").Result()
	if err != nil {
		panic(err)
	}
	if int(num) != n {
		t.Fatalf("Expected 100 clients, but got: %v", num)
	}
}

// TestConcurrentWrite tests writing to a specific client concurrently
func TestConcurrentWrite(t *testing.T) {
	n := 40
	// make 100 connections as write clients
	connections := makeWsConnections(t, n)

	// make one connection, which is all messages are sent to
	conn1, _, _, err := ws.DefaultDialer.Dial(context.Background(), WsServerAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer func() {
		conn1.Close()
		for i := 0; i < n; i++ {
			connections[i].conn.Close()
		}
	}()

	// msg is in the form of {"body": xxx}, where xxx is the connection id
	msg, _, err := wsutil.ReadServerData(conn1)
	if err != nil {
		t.Fatalf("Failed to read from server: %v", err)
	}
	id := msg[10 : len(msg)-2]

	// message to be sent
	body := make(map[string]interface{})
	body["receiver_ids"] = []string{string(id)}
	body["msg"] = "hello"
	message := WsPacket{Action: "run", Target: "sendMsg", Body: body}
	msgByte, err := json.Marshal(message)

	// send message via lambda function to conn1 concurrently
	for i := 0; i < n; i++ {
		i := i
		go func() {
			err := wsutil.WriteClientText(connections[i].conn, msgByte)
			if err != nil {
				t.Fatalf("Failed to write to server: %v", err)
			}
		}()
	}

	replies := make(chan Reply, n)
	go func() {
		for {
			replyBytes, _, err := wsutil.ReadServerData(conn1)
			if err != nil {
				break
			}
			var reply Reply
			err = json.Unmarshal(replyBytes, &reply)
			if err != nil {
				t.Fatalf("Failed to unmarshal reply: %v", err)
				return
			}
			replies <- reply
		}
	}()

	msgCount := 0
	duration := 100 * time.Second
	timeout := time.After(duration)
loop:
	for {
		select {
		case reply := <-replies:
			if reply.Body != body["msg"] {
				t.Errorf("Expected message: %v, but got: %v", body["msg"], reply.Body)
			}
			msgCount++
			fmt.Println(msgCount, "with", string(msg))
			if msgCount == n {
				break loop
			}
		case <-timeout:
			log.Fatal("Timeout after", duration)
		}
	}
}

// TestBroadcast tests broadcasting to all clients
func TestBroadcast(t *testing.T) {
	// make 100 connections as clients
	connections := makeWsConnections(t, 100)

	// message to be sent
	body := make(map[string]interface{})
	body["msg"] = "hello"
	message := WsPacket{Action: "run", Target: "broadcastMsg", Body: body}
	msgByte, err := json.Marshal(message)
	// told every client to broadcast
	for i := 0; i < 100; i++ {
		err = wsutil.WriteServerText(connections[i].conn, msgByte)
		if err != nil {
			t.Fatalf("Failed to write to server: %v", err)
		}
	}

	// check if all connections receive 100 messages from each other
	for i := 0; i < 100; i++ {
		connectionIDs := make(map[string]bool, 100)
		for i := 0; i < 100; i++ {
			connectionIDs[connections[i].id] = true
		}
		go func(i int) {
			duration := 2 * time.Second
			timeout := time.After(duration)
			for {
				select {
				case <-timeout:
					fmt.Println("Timeout after", duration)
					return
				default:
					msg, _, err := wsutil.ReadServerData(connections[i].conn)
					if err != nil {
						t.Errorf("Failed to read from server: %v", err)
						return
					}
					var reply Reply
					json.Unmarshal(msg, &reply)
					if reply.Body != body["msg"] && reply.Body != "messages sent" {
						t.Errorf("Expected message: %v, but got: %v", body, reply.Body)
						return
					}
					if _, ok := connectionIDs[reply.SenderId]; !ok {
						t.Errorf("sender id not found or already deleted: %v", reply.SenderId)
						return
					}
					delete(connectionIDs, reply.SenderId)
				}
				if len(connectionIDs) != 0 {
					t.Errorf("Expected to receive 100 messages, but got: %v", 100-len(connectionIDs))
					return
				}
			}
		}(i)
	}
}

func makeWsConnections(t *testing.T, n int) []client {
	// make n clients
	connections := make([]client, n)
	for i := 0; i < n; i++ {
		conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), WsServerAddr)
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		msg, _, err := wsutil.ReadServerData(conn)
		connections[i] = client{conn: conn, id: string(msg[8 : len(msg)-2])}
	}
	return connections
}

func TestPost(t *testing.T) {
	resps := make(chan *http.Response, 100)
	for i := 0; i < 100; i++ {
		go func() {
			body := make(map[string]interface{})
			body["receiver_ids"] = []string{uuid.New().String()}
			body["msg"] = "hello"

			e := Event{&Context{Id: uuid.New()}, &body}
			eventBytes, err := json.Marshal(e)
			if err != nil {
				fmt.Printf("Failed to marshal event: %v", err)
				return
			}

			resp, post := http.Post(
				"http://localhost:5000/run/sendMsg",
				"application/json",
				bytes.NewBuffer(eventBytes))
			if post != nil {
				return
			}
			resps <- resp
			defer resp.Body.Close()
		}()
	}

	cnt := 0
	for {
		select {
		case resp := <-resps:
			cnt++
			fmt.Println(cnt)
			var body []byte
			resp.Body.Read(body)
			fmt.Printf(string(body))
		}
	}
}
