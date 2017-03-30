package policy

import (
	"math/rand"
	"time"
)

type RandomEvictor struct {
}

func NewRandomEvictor() (re *RandomEvictor) {
	rand.Seed(time.Now().Unix())

	return &RandomEvictor{}
}

func (re *RandomEvictor) Evict(servers []ForkServer) error {
	_ = rand.Int() % len(servers)
	//TODO: actually evict

	return nil
}
