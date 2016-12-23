package registry

import (
	r "github.com/open-lambda/open-lambda/registry/src"
)

func Push(server_ip string, name string, fname string) {
	pushc := r.InitPushClient(server_ip, r.CHUNK_SIZE)
	handler := r.PushClientFile{Name: fname, Type: r.HANDLER}
	pushc.Push(name, handler)
}
