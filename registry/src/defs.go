package registry

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/open-lambda/registry/src/regproto"
	r "gopkg.in/dancannon/gorethink.v2"
)

type PushClient struct {
	ServerAddr string
	ChunkSize  int
	Conn       pb.RegistryClient
}

type PushClientFile struct {
	Name string
	Type string
}

type PushServer struct {
	Port      int
	ChunkSize int
	Conn      *r.Session // sessions are thread safe?
	Processor FileProcessor
}

// files: map of filetype -> file
type FileProcessor interface {
	Process(name string, files map[string][]byte) ([]DBInsert, error)
}

type DBInsert struct {
	Table string
	Data  *map[string]interface{}
}

type PullClient struct {
	Type  string
	Conn  *r.Session
	Table string
}

func grpcCheck(err error) {
	if err != nil {
		grpclog.Fatal(grpc.ErrorDesc(err))
	}
}
