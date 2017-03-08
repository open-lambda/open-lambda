package policy

import (
	"math/rand"
	"time"
)

type RandomEvictor struct {
	servers []ForkServer
}

func NewRandomEvictor(servers []ForkServer) (re *RandomEvictor) {
	rand.Seed(time.Now().Unix())

	return &RandomEvictor{
		servers: servers,
	}
}

func (re *RandomEvictor) Evict() error {
	_ = rand.Int() % len(re.servers)
	//TODO: actually evict

	return nil
}
