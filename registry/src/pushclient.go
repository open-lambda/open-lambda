package registry

import (
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/open-lambda/open-lambda/registry/src/regproto"
)

func (c *PushClient) sendFile(stream pb.Registry_PushClient, name string, file PushClientFile) {
	data, err := ioutil.ReadFile(file.Name)
	grpcCheck(err)
	/*
		n := bytes.IndexByte(data, 0)
		fmt.Printf(string(data[:]))
		fmt.Printf("%v\n", n)
		data = data[:n]
	*/
	r := bytes.NewReader(data)
	for {
		chunk := make([]byte, c.ChunkSize)
		n, err := r.Read(chunk)
		if n == 0 && err == io.EOF {
			return
		}

		err = stream.Send(&pb.Chunk{
			Name:     name,
			Data:     chunk[:n],
			FileType: file.Type,
		})
		grpcCheck(err)
	}

	return
}

func (c *PushClient) Push(name string, files ...PushClientFile) {
	stream, err := c.Conn.Push(context.Background())
	grpcCheck(err)

	for _, file := range files {
		c.sendFile(stream, name, file)
	}

	_, err = stream.CloseAndRecv()
	grpcCheck(err)

	return
}

func InitPushClient(serveraddr string, chunksize int) *PushClient {
	c := new(PushClient)

	c.ServerAddr = serveraddr
	c.ChunkSize = chunksize

	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(c.ServerAddr, opts...)
	grpcCheck(err)

	c.Conn = pb.NewRegistryClient(conn)

	return c
}
