package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	pb "github.com/open-lambda/open-lambda/ol/websocket/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

type WsManager struct {
	clients      sync.Map
	clientsEpoll *Epoll
	pb.UnimplementedWsManagerServer
}

func NewWsManager() *WsManager {
	epoll, _ := CreateEpoll(nil)
	return &WsManager{
		clients:      sync.Map{},
		clientsEpoll: epoll,
	}
}

// RegisterClient registers a client and its callback function to epoll, then call onConnect func in lambda
func (manager *WsManager) RegisterClient(client *Client) {
	manager.clients.Store(client.id, client)
	// register client to epoll
	manager.clientsEpoll.Add(client, EPOLLIN|EPOLLHUP,
		func(event EpollEvent) {
			op, r, err := wsutil.NextReader(client.conn, ws.StateServerSide)
			if err != nil {
				fmt.Println("error:", err)
				client.onDisconnect()
			}

			// Todo: handle ping/pong, and accidentally close
			if op.OpCode.IsControl() {
				switch op.OpCode {
				case ws.OpClose:
					client.onDisconnect()
				}
			}

			req := &(client.wsPacket)
			decoder := json.NewDecoder(r)
			if err := decoder.Decode(req); err != nil {
				err := fmt.Errorf("error parsing packet: %v", err)
				log.Println(err.Error())
				client.write([]byte(err.Error()))
				return
			}
			client.event = &Event{
				Context: &Context{Id: client.id},
				Body:    &req.Body,
			}
			go router(client)
		})
	// test number of elements in the map todo: delete this
	count := 0
	manager.clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	fmt.Println("Number of elements:", count)
}

func (manager *WsManager) UnregisterClient(client *Client) {
	id := client.id
	manager.clients.Delete(id)

	// unregister client from epoll
	manager.clientsEpoll.Remove(client)

}

func (manager *WsManager) GetClient(id uuid.UUID) (*Client, bool) {
	client, ok := manager.clients.Load(id)

	if ok {
		return client.(*Client), true
	}
	return nil, false
}

// startInternalApi starts internal APIs(grpc) for lambda functions
func (manager *WsManager) startInternalApi() {
	log.Println("internal APIs for lambda functions started")
	ip := "0.0.0.0"
	lis, err := net.Listen("tcp", ip+":50051")
	if err != nil {
		log.Fatalf("internal APIs failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWsManagerServer(s, manager)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("internal APIs failed to serve: %v", err)
	}
}

// PostToConnection is an internal gRPC method for lambda functions to call
// it will post message to a connection
func (manager *WsManager) PostToConnection(ctx context.Context, req *pb.PostToConnectionRequest) (*pb.PostToConnectionResponse, error) {
	// log.Println("posting to connection: ", req.ConnectionId)
	u, err := uuid.Parse(req.ConnectionId)
	if err != nil {
		return &pb.PostToConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse UUID: %s", err),
		}, nil
	}

	wsClient, ok := manager.GetClient(u)
	if !ok {
		log.Printf("Client:%v not found \n", u)
		return &pb.PostToConnectionResponse{
			Success: false,
			Error:   "Client not found",
		}, nil
	}

	wsClient.writeMux.Lock()
	err = wsutil.WriteServerText(wsClient.conn, []byte(req.Msg))
	wsClient.writeMux.Unlock()

	if err != nil {
		return &pb.PostToConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to write WebSocket message: %s", err),
		}, nil
	}
	return &pb.PostToConnectionResponse{
		Success: true,
		Error:   "",
	}, nil
}
