package olreg

import (
	r "github.com/open-lambda/code-registry/registry"
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
