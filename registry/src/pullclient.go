package olreg

import r "github.com/open-lambda/code-registry/registry"

func InitPullClient(cluster []string) *PullClient {
	c := PullClient{
		Client: r.InitPullClient(cluster, DATABASE, TABLE),
	}

	return &c
}

func (c *PullClient) Pull(name string) []byte {
	files := c.Client.Pull(name)

	handler := files[HANDLER].([]byte)

	return handler
}
