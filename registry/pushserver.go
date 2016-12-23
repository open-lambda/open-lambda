package registry

import (
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
