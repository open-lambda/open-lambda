package lambdastore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/boss/event"
	"github.com/open-lambda/open-lambda/ol/common"
)

type LambdaStore struct {
	// bucket is the cloud storage bucket for lambda tarballs
	bucket *blob.Bucket

	// The tarball is atomically moved here while holding both mapLock and the lambda’s entry.Lock,
	// ensuring consistency between in-memory and on-disk state. Actual deletion from disk
	// is deferred to a background goroutine to avoid holding locks during slow I/O.

	eventManager *event.Manager
	// mapLock protects concurrent access to the Lambdas map
	mapLock sync.Mutex
	Lambdas map[string]*LambdaEntry
}

type LambdaEntry struct {
	Config *common.LambdaConfig
	Lock   *sync.Mutex
}

func NewLambdaStore(storeURL string, pool *cloudvm.WorkerPool) (*LambdaStore, error) {
	ctx := context.Background()

	// If using local file storage, ensure the directory exists
	if strings.HasPrefix(storeURL, "file://") {
		dir := strings.TrimPrefix(storeURL, "file://")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local lambda store directory %s: %w", dir, err)
		}
	}

	bucket, err := blob.OpenBucket(ctx, storeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob bucket: %w", err)
	}

	var eventManager *event.Manager
	if pool != nil {
		eventManager = event.NewManager(pool)
	}

	store := &LambdaStore{
		StorePath:    storePath,
		trashPath:    trashDir,
		eventManager: eventManager,
		Lambdas:      make(map[string]*LambdaEntry),
	}

	// Load existing lambdas by listing objects in the bucket
	iter := bucket.List(&blob.ListOptions{})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list bucket objects: %w", err)
		}
		if strings.HasSuffix(obj.Key, ".tar.gz") {
			funcName := strings.TrimSuffix(obj.Key, ".tar.gz")
			if err := store.loadConfigAndRegister(funcName); err != nil {
				log.Printf("Failed to load lambda %s: %v", funcName, err)
			}
		}
	}

	return store, nil
}

// ------------------- HTTP Handlers ----------------------

func (s *LambdaStore) UploadLambda(w http.ResponseWriter, r *http.Request) {
	funcName := strings.TrimPrefix(r.URL.Path, "/registry/")

	if err := common.ValidateFunctionName(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.addToRegistry(funcName, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add lambda: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded successfully", funcName)
}

func (s *LambdaStore) DeleteLambda(w http.ResponseWriter, r *http.Request) {
	funcName := strings.TrimPrefix(r.URL.Path, "/registry/")

	if err := common.ValidateFunctionName(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.removeFromRegistry(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", funcName)
}

func (s *LambdaStore) ListLambda(w http.ResponseWriter) {
	funcNames := s.listEntries()

	if err := json.NewEncoder(w).Encode(funcNames); err != nil {
		http.Error(w, "failed to encode lambda list", http.StatusInternalServerError)
	}
}

func (s *LambdaStore) RetrieveLambdaConfig(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/registry/")
	parts := strings.SplitN(raw, "/", 2)

	if len(parts) != 2 || parts[1] != "config" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	funcName := parts[0]

	if err := common.ValidateFunctionName(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, err := s.getConfig(funcName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, "failed to encode config as JSON", http.StatusInternalServerError)
		return
	}
}

// ------------------- Core Logic ----------------------

func (s *LambdaStore) loadConfigAndRegister(funcName string) error {
	ctx := context.Background()
	key := funcName + ".tar.gz"

	// Read the tarball from blob storage
	reader, err := s.bucket.NewReader(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("failed to open blob reader: %w", err)
	}
	defer reader.Close()

	// Download to a temp file for config extraction
	tempFile, err := os.CreateTemp("", funcName+"_*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, reader); err != nil {
		return fmt.Errorf("failed to download blob: %w", err)
	}

	cfg, err := common.ExtractConfigFromTarGz(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to extract config.json: %w", err)
	}

	entry := s.getOrCreateEntry(funcName)
	entry.Lock.Lock()
	defer entry.Lock.Unlock()

	entry.Config = cfg

	if s.eventManager != nil {
		err = s.eventManager.Register(funcName, cfg.Triggers)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *LambdaStore) addToRegistry(funcName string, body io.Reader) error {
	lambdaEntry := s.getOrCreateEntry(funcName)
	lambdaEntry.Lock.Lock()
	defer lambdaEntry.Lock.Unlock()

	ctx := context.Background()
	key := funcName + ".tar.gz"

	// Create a temporary file to validate the tarball
	tempFile, err := os.CreateTemp("", funcName+"_upload_*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp tarball: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, body); err != nil {
		return fmt.Errorf("failed to write to temp tarball: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp tarball: %w", err)
	}

	// Extract config to validate the tarball
	cfg, err := common.ExtractConfigFromTarGz(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to extract config from temp tarball: %w", err)
	}

	// Upload to blob storage
	tempFile, err = os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to reopen temp file: %w", err)
	}
	defer tempFile.Close()

	writer, err := s.bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("failed to create blob writer: %w", err)
	}

	if _, err := io.Copy(writer, tempFile); err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload to blob storage: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close blob writer: %w", err)
	}

	lambdaEntry.Config = cfg

	if s.eventManager != nil {
		err = s.eventManager.Register(funcName, cfg.Triggers)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *LambdaStore) removeFromRegistry(funcName string) error {
	s.mapLock.Lock()
	entry, ok := s.Lambdas[funcName]
	if !ok {
		s.mapLock.Unlock()
		return fmt.Errorf("lambda %s not found", funcName)
	}
	// hold both locks
	entry.Lock.Lock()

	ctx := context.Background()
	key := funcName + ".tar.gz"

	if s.eventManager != nil {
		if err := s.eventManager.Unregister(funcName); err != nil {
			log.Printf("failed to unregister triggers for %s: %v", funcName, err)
		}
	}

	delete(s.Lambdas, funcName)
	entry.Lock.Unlock()
	s.mapLock.Unlock()

	// Background deletion from blob storage
	go func() {
		if err := s.bucket.Delete(ctx, key); err != nil {
			log.Printf("warning: failed to delete %s from blob storage: %v", key, err)
		}
	}()

	return nil
}

func (s *LambdaStore) getConfig(funcName string) (*common.LambdaConfig, error) {
	lambdaEntry := s.getOrCreateEntry(funcName)
	lambdaEntry.Lock.Lock()
	defer lambdaEntry.Lock.Unlock()

	// If we already have the config cached, return it
	if lambdaEntry.Config != nil {
		return lambdaEntry.Config, nil
	}

	ctx := context.Background()
	key := funcName + ".tar.gz"

	// Download from blob storage to temp file
	reader, err := s.bucket.NewReader(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("lambda %q not found", funcName)
	}
	defer reader.Close()

	tempFile, err := os.CreateTemp("", funcName+"_config_*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, reader); err != nil {
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	cfg, err := common.ExtractConfigFromTarGz(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to extract config: %w", err)
	}

	return cfg, nil
}

func (s *LambdaStore) getOrCreateEntry(funcName string) *LambdaEntry {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	entry, ok := s.Lambdas[funcName]
	if !ok {
		entry = &LambdaEntry{
			Lock:   &sync.Mutex{},
			Config: nil,
		}
		s.Lambdas[funcName] = entry
	}

	return entry
}

func (s *LambdaStore) listEntries() []string {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	funcNames := make([]string, 0, len(s.Lambdas))
	for name := range s.Lambdas {
		funcNames = append(funcNames, name)
	}
	return funcNames
}
