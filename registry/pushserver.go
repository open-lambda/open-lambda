// +build pushserver

package main

import (
	"log"
	"os"
	"strconv"

	r "github.com/open-lambda/open-lambda/registry/src"
)

type FileProcessor struct{}

func (p FileProcessor) Process(name string, files map[string][]byte) ([]r.DBInsert, error) {
	ret := make([]r.DBInsert, 0)
	f := map[string]interface{}{
		"id":      name,
		"handler": files[r.HANDLER],
	}
	insert := r.DBInsert{
		Table: r.TABLE,
		Data:  &f,
	}
	ret = append(ret, insert)

	return ret, nil
}

func InitPushServer(port int, cluster []string) *r.PushServer {
	proc := FileProcessor{}
	return r.InitPushServer(cluster, r.DATABASE, proc, port, r.CHUNK_SIZE, r.TABLE)
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
