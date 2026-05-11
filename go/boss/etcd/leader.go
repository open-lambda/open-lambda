package etcd

import (
	"context"
	"fmt"
	"sync"

	"go.etcd.io/etcd/client/v3/concurrency"
)

// manages this boss instance's participation in the etcd election.
// only one boss is leader at a time - followers redirect mutating requests to leader
type LeaderElection struct {
	session  *concurrency.Session
	election *concurrency.Election
	mu       sync.RWMutex
	leader   bool
}

// creates an election participant backed by a TTL session.
// if this boss dies, the session expires and the next candidate wins within TTL
func (c *Client) NewLeaderElection(ctx context.Context) (*LeaderElection, error) {
	session, err := concurrency.NewSession(c.raw, concurrency.WithTTL(10))
	if err != nil {
		return nil, fmt.Errorf("etcd session: %w", err)
	}
	return &LeaderElection{
		session:  session,
		election: concurrency.NewElection(session, c.key("boss", "leader")),
	}, nil
}

// blocks until this instance wins the election or ctx is cancelled.
// selfAddr ("host:port") is stored as the election value so followers can
// redirect requests to the leader.
func (le *LeaderElection) Campaign(ctx context.Context, selfAddr string) error {
	if err := le.election.Campaign(ctx, selfAddr); err != nil {
		return err
	}
	le.mu.Lock()
	le.leader = true
	le.mu.Unlock()
	return nil
}

// yields leadership immediately so the next candidate wins without
// waiting for the TTL to expire. should be called on clean shutdown.
func (le *LeaderElection) Resign(ctx context.Context) error {
	le.mu.Lock()
	le.leader = false
	le.mu.Unlock()
	return le.election.Resign(ctx)
}

func (le *LeaderElection) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.leader
}

// returns the current leader "host:port" by reading the election key.
func (le *LeaderElection) LeaderAddr(ctx context.Context) (string, error) {
	resp, err := le.election.Leader(ctx)
	if err != nil {
		return "", fmt.Errorf("etcd leader lookup: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no leader elected yet")
	}
	return string(resp.Kvs[0].Value), nil
}

func (le *LeaderElection) Close() error {
	return le.session.Close()
}
