package etcd

import (
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// client wraps the etcd v3 client and owns the connection lifecycle.
// all key-space operations are namespaced under a configurable prefix.
type Client struct {
	raw    *clientv3.Client
	kv     clientv3.KV
	prefix string
}

func NewClient(endpoints []string, prefix string, dialTimeout time.Duration) (*Client, error) {
	raw, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd dial failed: %w", err)
	}
	return &Client{
		raw:    raw,
		kv:     clientv3.NewKV(raw),
		prefix: strings.TrimRight(prefix, "/"),
	}, nil
}

func (c *Client) Close() error {
	return c.raw.Close()
}

// key returns a namespaced etcd key.
func (c *Client) key(parts ...string) string {
	return c.prefix + "/" + strings.Join(parts, "/")
}
