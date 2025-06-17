package lambdastore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/open-lambda/open-lambda/ol/common"
)

type GCSLambdaStore struct {
	BucketName string
	Prefix     string
	Client     *storage.Client
	Lambdas    map[string]*LambdaEntry
	mapLock    sync.Mutex
}

func NewGCSLambdaStore(bucket, prefix string) (*GCSLambdaStore, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	store := &GCSLambdaStore{
		BucketName: bucket,
		Prefix:     prefix,
		Client:     client,
		Lambdas:    make(map[string]*LambdaEntry),
	}

	// Optionally preload known lambdas
	return store, nil
}

func (s *GCSLambdaStore) UploadLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/registry/")
	if err := common.ValidateFunctionName(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lock := s.getFuncLock(functionName)
	lock.Lock()
	defer lock.Unlock()

	if err := s.removeFromRegistry(functionName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to clean old version: %v", err), http.StatusInternalServerError)
		return
	}
	if err := s.addToRegistry(functionName, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add lambda: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded successfully", functionName)
}

func (s *GCSLambdaStore) DeleteLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/registry/")
	if err := common.ValidateFunctionName(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lock := s.getFuncLock(functionName)
	lock.Lock()
	defer lock.Unlock()

	if err := s.removeFromRegistry(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", functionName)
}

func (s *GCSLambdaStore) ListLambda(w http.ResponseWriter, r *http.Request) {
	s.mapLock.Lock()
	names := make([]string, 0, len(s.Lambdas))
	for name := range s.Lambdas {
		names = append(names, name)
	}
	s.mapLock.Unlock()

	if err := json.NewEncoder(w).Encode(names); err != nil {
		http.Error(w, "failed to encode lambda list", http.StatusInternalServerError)
	}
}

func (s *GCSLambdaStore) GetLambdaConfig(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/registry/")
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) != 2 || parts[1] != "config" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	functionName := parts[0]
	if err := common.ValidateFunctionName(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lock := s.getFuncLock(functionName)
	lock.Lock()
	defer lock.Unlock()

	r, err := s.readTarball(functionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read tarball: %v", err), http.StatusInternalServerError)
		return
	}
	cfg, err := common.ExtractConfigFromTarGzStream(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to extract config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, "failed to encode config as JSON", http.StatusInternalServerError)
	}
}

func (s *GCSLambdaStore) addToRegistry(name string, body io.Reader) error {
	ctx := context.Background()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, body); err != nil {
		return err
	}

	// Validate tarball by extracting config
	cfg, err := common.ExtractConfigFromTarGzStream(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("invalid tar.gz: %w", err)
	}

	wc := s.Client.Bucket(s.BucketName).Object(s.Prefix + name + ".tar.gz").NewWriter(ctx)
	wc.ContentType = "application/gzip"
	wc.CacheControl = "no-cache"
	if _, err := io.Copy(wc, bytes.NewReader(buf.Bytes())); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	if err := wc.Close(); err != nil {
		return err
	}

	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	entry, ok := s.Lambdas[name]
	if !ok {
		entry = &LambdaEntry{Lock: &sync.Mutex{}}
		s.Lambdas[name] = entry
	}
	entry.Config = cfg
	return nil
}

func (s *GCSLambdaStore) removeFromRegistry(name string) error {
	ctx := context.Background()
	obj := s.Client.Bucket(s.BucketName).Object(s.Prefix + name + ".tar.gz")
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete from GCS: %w", err)
	}

	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	delete(s.Lambdas, name)
	return nil
}

func (s *GCSLambdaStore) getFuncLock(name string) *sync.Mutex {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	entry, ok := s.Lambdas[name]
	if !ok {
		entry = &LambdaEntry{Lock: &sync.Mutex{}, Config: nil}
		s.Lambdas[name] = entry
	}
	return entry.Lock
}

func (s *GCSLambdaStore) readTarball(name string) (io.Reader, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := s.Client.Bucket(s.BucketName).Object(s.Prefix + name + ".tar.gz").NewReader(ctx)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}
	r.Close()
	return bytes.NewReader(buf.Bytes()), nil
}
