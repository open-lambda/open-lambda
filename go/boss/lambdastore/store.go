package lambdastore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/open-lambda/open-lambda/go/boss/cloudvm"
	"github.com/open-lambda/open-lambda/go/boss/event"
	"github.com/open-lambda/open-lambda/go/common"
)

type LambdaStore struct {
	// bucket is the cloud storage bucket for lambda tarballs
	bucket *blob.Bucket

	eventManager *event.Manager
	// mapLock protects concurrent access to the Lambdas map
	mapLock sync.Mutex
	Lambdas map[string]*LambdaEntry
}

type LambdaEntry struct {
	Config *common.LambdaConfig
	Lock   *sync.Mutex
}

// NewLambdaStore creates a new lambda store backed by cloud storage.
// pool may be nil or a WorkerPool instance depending on the calling context:
//   - Boss context (pool provided): Full functionality including lambda registry and event-driven execution
//     (cron triggers, Kafka triggers). Called from boss.go:156 with a worker pool.
//   - Worker context (pool is nil): Limited functionality with lambda registry only (upload, delete, list, config).
//     Event-driven execution is disabled. Called from worker/event/server.go:282 with nil pool.
func NewLambdaStore(storeURL string, pool *cloudvm.WorkerPool) (*LambdaStore, error) {
	ctx := context.Background()

	// If no recognized scheme is present, assume local path and add file://
	if !strings.HasPrefix(storeURL, "file://") &&
		!strings.HasPrefix(storeURL, "s3://") &&
		!strings.HasPrefix(storeURL, "gs://") {
		storeURL = "file://" + storeURL
	}

	// If using local file storage, ensure the directory exists and configure
	// fileblob to create temp files in the same directory as the target.
	// This avoids "invalid cross-device link" errors when /tmp is on a
	// different filesystem/mount than the registry directory.
	if strings.HasPrefix(storeURL, "file://") {
		dir := strings.TrimPrefix(storeURL, "file://")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local lambda store directory %s: %w", dir, err)
		}
		storeURL = storeURL + "?no_tmp_dir=true"
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
		bucket:       bucket,
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
		if strings.HasSuffix(obj.Key, common.LambdaFileExtension) {
			funcName := strings.TrimSuffix(obj.Key, common.LambdaFileExtension)
			if err := store.loadConfigAndRegister(funcName); err != nil {
				slog.Error(fmt.Sprintf("Failed to load lambda %s: %v", funcName, err))
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
	funcNames := s.ListEntries()

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

	cfg, err := s.GetConfig(funcName)
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
	key := funcName + common.LambdaFileExtension

	// Read the tarball from blob storage
	reader, err := s.bucket.NewReader(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("failed to open blob reader: %w", err)
	}
	defer reader.Close()

	// Download to a temp file for config extraction
	tempFile, err := os.CreateTemp("", funcName+"_*"+common.LambdaFileExtension)
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
	key := funcName + common.LambdaFileExtension

	// Create a temporary file to validate the tarball
	tempFile, err := os.CreateTemp("", funcName+"_upload_*"+common.LambdaFileExtension)
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
		// Close writer to release resources (ignore close error since we already have an error)
		writer.Close()
		return fmt.Errorf("failed to upload to blob storage: %w", err)
	}

	// Close the writer to finalize the upload - this is where the actual commit happens
	// for many blob storage implementations, so we must check the error
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize blob upload: %w", err)
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

	if s.eventManager != nil {
		if err := s.eventManager.Unregister(funcName); err != nil {
			slog.Error(fmt.Sprintf("failed to unregister triggers for %s: %v", funcName, err))
		}
	}

	delete(s.Lambdas, funcName)
	entry.Lock.Unlock()
	s.mapLock.Unlock()

	// Background deletion
	go func() {
		if err := s.bucket.Delete(context.Background(), funcName+common.LambdaFileExtension); err != nil {
			slog.Error(fmt.Sprintf("warning: failed to remove %s from blob storage: %v", funcName+common.LambdaFileExtension, err))
		}
	}()

	return nil
}

func (s *LambdaStore) GetConfig(funcName string) (*common.LambdaConfig, error) {
	lambdaEntry := s.getOrCreateEntry(funcName)
	lambdaEntry.Lock.Lock()
	defer lambdaEntry.Lock.Unlock()

	// Return cached config if available
	if lambdaEntry.Config != nil {
		return lambdaEntry.Config, nil
	}

	// If not cached, try to load from blob storage
	if err := s.loadConfigAndRegister(funcName); err != nil {
		return nil, fmt.Errorf("failed to load lambda %q: %w", funcName, err)
	}

	return lambdaEntry.Config, nil
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

func (s *LambdaStore) ListEntries() []string {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	funcNames := make([]string, 0, len(s.Lambdas))
	for name := range s.Lambdas {
		funcNames = append(funcNames, name)
	}
	return funcNames
}
