package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// WorkerRecord is the etcd-serialized snapshot of a single worker.
// State mirrors the cloudvm WorkerState constants (0=STARTING … 3=DESTROYING).
type WorkerRecord struct {
	WorkerId string `json:"worker_id"`
	State    int    `json:"state"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

// PoolMeta holds pool-level counters that must survive boss restarts.
type PoolMeta struct {
	NextId int `json:"next_id"` // next worker ID to assign; prevents ID reuse
	Target int `json:"target"`  // desired number of running workers
}

func (c *Client) workerKey(workerId string) string {
	return c.key("workers", workerId)
}

func (c *Client) poolMetaKey() string {
	return c.key("pool", "meta")
}

func (c *Client) PutWorker(ctx context.Context, rec WorkerRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal worker record: %w", err)
	}
	_, err = c.kv.Put(ctx, c.workerKey(rec.WorkerId), string(b))
	return err
}

func (c *Client) DeleteWorker(ctx context.Context, workerId string) error {
	_, err := c.kv.Delete(ctx, c.workerKey(workerId))
	return err
}

// RestoreWorkers fetches all worker records stored under the workers/ prefix.
func (c *Client) RestoreWorkers(ctx context.Context) ([]WorkerRecord, error) {
	prefix := c.key("workers") + "/"
	resp, err := c.kv.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd get workers: %w", err)
	}
	records := make([]WorkerRecord, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var rec WorkerRecord
		if err := json.Unmarshal(kv.Value, &rec); err != nil {
			slog.Warn("skipping unreadable worker record", "key", string(kv.Key), "err", err)
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

func (c *Client) PutPoolMeta(ctx context.Context, meta PoolMeta) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal pool meta: %w", err)
	}
	_, err = c.kv.Put(ctx, c.poolMetaKey(), string(b))
	return err
}

// GetPoolMeta returns the stored pool metadata. The bool is false when no
// metadata has been written yet (fresh etcd, first boss start).
func (c *Client) GetPoolMeta(ctx context.Context) (PoolMeta, bool, error) {
	resp, err := c.kv.Get(ctx, c.poolMetaKey())
	if err != nil {
		return PoolMeta{}, false, fmt.Errorf("etcd get pool meta: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return PoolMeta{}, false, nil
	}
	var meta PoolMeta
	if err := json.Unmarshal(resp.Kvs[0].Value, &meta); err != nil {
		return PoolMeta{}, false, fmt.Errorf("unmarshal pool meta: %w", err)
	}
	return meta, true, nil
}
