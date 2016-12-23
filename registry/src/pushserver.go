package registry

import (
	"fmt"
	"io"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/open-lambda/registry/src/regproto"
	r "gopkg.in/dancannon/gorethink.v2"
)

func (s *PushServer) processAndStore(name string, files map[string][]byte) error {
	procfiles, err := s.Processor.Process(name, files)
	if err != nil {
		return err
	}

	opts := r.InsertOpts{Conflict: "replace"}
	for _, file := range procfiles {
		_, err := r.Table(file.Table).Insert(file.Data, opts).RunWrite(s.Conn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PushServer) Push(stream pb.Registry_PushServer) error {
	files := make(map[string][]byte)
	name := ""

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			if name == "" {
				grpclog.Fatal("Empty push request or name field")
			}

			log.Printf("Save %v chunk(s)\n", name)
			err = s.processAndStore(name, files)
			if err != nil {
				return err
			}

			return stream.SendAndClose(&pb.Received{
				Received: true,
			})
		}

		ftype := chunk.FileType
		if val, ok := files[ftype]; ok {
			files[ftype] = append(val, chunk.Data...)
		} else {
			files[ftype] = chunk.Data
		}

		name = chunk.Name
		grpcCheck(err)
	}

	return nil
}

func (s *PushServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	grpcCheck(err)

	grpcServer := grpc.NewServer()
	pb.RegisterRegistryServer(grpcServer, s)
	grpcServer.Serve(lis)

	return
}

// TODO add authKey argument to creating the session
func InitPushServer(cluster []string, db string, proc FileProcessor, port, chunksize int, tables ...string) *PushServer {
	s := new(PushServer)

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  db,
	})
	grpcCheck(err)

	_, _ = r.DBCreate(db).RunWrite(session)
	for _, table := range tables {
		_, _ = r.TableCreate(table).RunWrite(session)
	}

	s.Conn = session
	s.Port = port
	s.ChunkSize = chunksize
	s.Processor = proc

	return s
}
