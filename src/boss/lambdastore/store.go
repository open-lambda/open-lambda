package lambdastore

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/boss/event"
	"github.com/open-lambda/open-lambda/ol/common"
)

type LambdaStore struct {
	StorePath    string
	trashPath    string
	eventManager *event.Manager
	// mapLock protects concurrent access to the Lambdas map
	mapLock sync.Mutex
	Lambdas map[string]*LambdaEntry
}

type LambdaEntry struct {
	Config *common.LambdaConfig
	Lock   *sync.Mutex
}

func NewLambdaStore(storePath string, pool *cloudvm.WorkerPool) (*LambdaStore, error) {
	trashDir := filepath.Join(storePath, ".trash")

	store := &LambdaStore{
		StorePath:    storePath,
		trashPath:    trashDir,
		eventManager: event.NewManager(pool),
		Lambdas:      make(map[string]*LambdaEntry),
	}

	if err := os.MkdirAll(store.StorePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lambda store directory: %w", err)
	}

	// Ensure the .trash directory exists for safe async deletion
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .trash directory: %w", err)
	}

	files, err := os.ReadDir(store.StorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lambda store directory: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.gz") {
			funcName := strings.TrimSuffix(file.Name(), ".tar.gz")
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
	tarPath := filepath.Join(s.StorePath, funcName+".tar.gz")

	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		return fmt.Errorf("failed to extract config.json: %w", err)
	}

	entry := s.getOrCreateEntry(funcName)

	entry.Lock.Lock()
	entry.Config = cfg
	entry.Lock.Unlock()

	err = s.eventManager.Register(funcName, cfg.Triggers)
	if err != nil {
		return err
	}

	return nil
}

func (s *LambdaStore) addToRegistry(funcName string, body io.Reader) error {
	lambdaEntry := s.getOrCreateEntry(funcName)
	lambdaEntry.Lock.Lock()
	defer lambdaEntry.Lock.Unlock()

	tarPath := filepath.Join(s.StorePath, funcName+".tar.gz")
	tmpPath := tarPath + ".tmp"

	tarFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp tarball: %w", err)
	}

	// always try to clean up temp file on return
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tarFile, body); err != nil {
		tarFile.Close()
		return fmt.Errorf("failed to write to temp tarball: %w", err)
	}

	if err := tarFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp tarball: %w", err)
	}

	cfg, err := common.ExtractConfigFromTarGz(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to extract config from temp tarball: %w", err)
	}

	// Atomically replace the old file
	if err := os.Rename(tmpPath, tarPath); err != nil {
		return fmt.Errorf("failed to rename temp tarball: %w", err)
	}

	lambdaEntry.Config = cfg

	err = s.eventManager.Register(funcName, cfg.Triggers)
	if err != nil {
		return err
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

	tarPath := filepath.Join(s.StorePath, funcName+".tar.gz")
	trashPath := filepath.Join(s.trashPath, funcName+".tar.gz")

	// Rename the file (fast + atomic)
	if err := os.Rename(tarPath, trashPath); err != nil && !os.IsNotExist(err) {
		entry.Lock.Unlock()
		s.mapLock.Unlock()
		return fmt.Errorf("failed to rename tarball for %s: %w", funcName, err)
	}

	delete(s.Lambdas, funcName)
	entry.Lock.Unlock()
	s.mapLock.Unlock()

	// Background deletion
	go func() {
		if err := os.Remove(trashPath); err != nil {
			log.Printf("warning: failed to remove %s from trash: %v", trashPath, err)
		}
	}()

	s.eventManager.Unregister(funcName)
	return nil
}

func (s *LambdaStore) getConfig(funcName string) (*common.LambdaConfig, error) {
	lambdaEntry := s.getOrCreateEntry(funcName)
	lambdaEntry.Lock.Lock()
	defer lambdaEntry.Lock.Unlock()

	tarPath := filepath.Join(s.StorePath, funcName+".tar.gz")
	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("lambda %q not found", funcName)
		}
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
