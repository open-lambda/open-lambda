package loadbalancer

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"sync"
)

var (
	hasher      hash.Hash
	hasherMutex sync.Mutex
)

func hashString(input int) int {
	hasherMutex.Lock()         // Lock the mutex before using the hasher
	defer hasherMutex.Unlock() // Unlock the mutex when the function exits

	hasher.Reset()
	buf := make([]byte, binary.MaxVarintLen64)
	binary.LittleEndian.PutUint64(buf, uint64(input))

	sum := sha256.Sum256(buf)
	// Take the first few bytes to fit into an int, ensuring it's always positive
	var truncatedHash int
	if size := binary.Size(truncatedHash); size == 64/8 {
		// 64-bit architecture
		truncatedHash = int(binary.LittleEndian.Uint64(sum[:8]) &^ (1 << 63))
	} else {
		// 32-bit architecture
		truncatedHash = int(binary.LittleEndian.Uint32(sum[:4]) &^ (1 << 31))
	}

	return truncatedHash
}

func HashGetGroup(pkgs []string, running int) int {
	node := root.Lookup(pkgs)
	hashInt := hashString(node.SplitGeneration)
	group := hashInt % running
	return group
}

func initHasher() {
	hasher = sha256.New()
}
