// +build pushserver

package main

import (
	"log"
	"os"
	"strconv"

	r "github.com/open-lambda/open-lambda/registry/src"
)

const (
	CHUNK_SIZE = 1024
	DATABASE   = "olregistry"
	HANDLER    = "handler"
	TABLE      = "handlers"
)

type FileProcessor struct{}

func (p FileProcessor) Process(name string, files map[string][]byte) ([]r.DBInsert, error) {
	ret := make([]r.DBInsert, 0)
	f := map[string]interface{}{
		"id":      name,
		"handler": files[HANDLER],
	}
	insert := r.DBInsert{
		Table: TABLE,
		Data:  &f,
	}
	ret = append(ret, insert)

	return ret, nil
}

func InitPushServer(port int, cluster []string) *r.PushServer {
	proc := FileProcessor{}
	return r.InitPushServer(cluster, DATABASE, proc, port, CHUNK_SIZE, TABLE)
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: pushserver <port> <cluster_ip1> <cluster_ip2>...")
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	pushs := InitPushServer(port, os.Args[2:])
	pushs.Run()
}
