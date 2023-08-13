package websocket

import "log"

// router parse the request and call the corresponding handler
func router(client *Client) {
	packet := client.wsPacket
	handler, ok := routeMap[packet.Action]
	if !ok {
		log.Println("Unknown action:", packet.Action)
		return
	}
	handler(client)
}

var routeMap = map[string]HandlerFunc{
	// todo: might contain other action, e.g pub/sub, in the future
	"run": func(v interface{}) { run(v.(*Client)) }, // run the lambda function specified in the target
}

func run(client *Client) {
	client.run()
}
