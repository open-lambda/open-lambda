package olreg

import r "github.com/open-lambda/code-registry/registry"

func InitPushClient(saddr string) *PushClient {
	c := r.InitPushClient(saddr, CHUNK_SIZE)

	return &PushClient{Client: c}
}

func (c *PushClient) PushFiles(name, fname, ftype string) {
	handler := r.PushClientFile{Name: fname, Type: ftype}

	c.Client.Push(name, handler)

	return
}
