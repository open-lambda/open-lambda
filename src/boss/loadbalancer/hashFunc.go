package loadbalancer

import (
	"crypto/sha256"
	"math/big"
)

func hashString(input string) int {
	hasherMutex.Lock()         // Lock the mutex before using the hasher
	defer hasherMutex.Unlock() // Unlock the mutex when the function exits

	hasher.Reset()
	hasher.Write([]byte(input))
	hashBytes := hasher.Sum(nil)

	bigIntHash := new(big.Int).SetBytes(hashBytes).Int64()
	if bigIntHash < 0 {
		bigIntHash = -bigIntHash
	}
	return int(bigIntHash)
}

func HashFuncGetGroup(img string, running int) int {
	hashInt := hashString(img)
	group := hashInt % running
	return group
}

func initFuncHasher() {
	hasher = sha256.New()
}
