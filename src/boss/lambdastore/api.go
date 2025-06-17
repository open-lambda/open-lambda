package lambdastore

import (
	"net/http"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

// LambdaStore defines the interface for managing lambdas across storage backends
// such as local filesystem or cloud storage (GCS, S3).
type LambdaStorePlatform interface {
	UploadLambda(w http.ResponseWriter, r *http.Request)
	DeleteLambda(w http.ResponseWriter, r *http.Request)
	ListLambda(w http.ResponseWriter, r *http.Request)
	GetLambdaConfig(w http.ResponseWriter, r *http.Request)

	// // Internal methods
	// addToRegistry(name string, body io.Reader) error
	// removeFromRegistry(name string) error
	// loadConfigAndRegister(name string) error
	// getFuncLock(name string) *sync.Mutex
}

// LambdaStore is a concrete type embedding a LambdaStorePlatform implementation.
// Allows room for common metadata, logging, stats, etc.
type LambdaStore struct {
	LambdaStorePlatform

	Platform string // "local", "gcp", etc.
}

// LambdaEntry holds metadata and lock for a single lambda
// It can be reused in both local and GCS implementations
type LambdaEntry struct {
	Config *common.LambdaConfig
	Lock   *sync.Mutex
}
