package registry

import (
	"log"

	r "gopkg.in/dancannon/gorethink.v2"
)

func (c *PullClient) Pull(name string) map[string]interface{} {
	ret := make(map[string]interface{})

	res, err := r.Table(c.Table).Get(name).Run(c.Conn)
	check(err)

	res.One(&ret)
	check(res.Err())

	return ret
}

func InitPullClient(cluster []string, db string, table string) *PullClient {
	c := new(PullClient)
	c.Table = table

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  db,
	})
	check(err)

	c.Conn = session

	return c
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
