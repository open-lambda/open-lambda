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

	"github.com/open-lambda/open-lambda/ol/boss/event"
	"github.com/open-lambda/open-lambda/ol/common"
)

type LambdaStore struct {
	StorePath string
	events    *event.Manager
	// mapLock protects concurrent access to the Lambdas map
	mapLock sync.Mutex
	Lambdas map[string]*LambdaEntry
}

type LambdaEntry struct {
	Config *common.LambdaConfig
	Lock   *sync.Mutex
}

func NewLambdaStore(storePath string) (*LambdaStore, error) {
	store := &LambdaStore{
		StorePath: storePath,
		events:    event.NewManager(),
		Lambdas:   make(map[string]*LambdaEntry),
	}

	if err := os.MkdirAll(store.StorePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lambda store directory: %w", err)
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
	functionName := strings.TrimPrefix(r.URL.Path, "/registry/")

	if err := common.ValidateFunctionName(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lock := s.getFuncLock(functionName)
	lock.Lock()
	defer lock.Unlock()

	// TODO: do not leave a window of time when no function exists betweem remove and add. Because workers don't sync with the boss, so they could have failed invocations.
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

func (s *LambdaStore) DeleteLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/registry/")

	if err := common.ValidateFunctionName(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	functionLock := s.getFuncLock(functionName)
	functionLock.Lock()
	defer functionLock.Unlock()

	if err := s.removeFromRegistry(functionName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", functionName)
}

func (s *LambdaStore) ListLambda(w http.ResponseWriter, r *http.Request) {
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

func (s *LambdaStore) GetLambdaConfig(w http.ResponseWriter, r *http.Request) {
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

	// protect against reading file during delete or upload
	functionLock := s.getFuncLock(functionName)
	functionLock.Lock()
	defer functionLock.Unlock()

	tarPath := filepath.Join(s.StorePath, functionName+".tar.gz")

	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to extract config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, "failed to encode config as JSON", http.StatusInternalServerError)
		return
	}
}

// ------------------- Core Logic ----------------------

func (s *LambdaStore) loadConfigAndRegister(functionName string) error {
	tarPath := filepath.Join(s.StorePath, functionName+".tar.gz")

	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		return fmt.Errorf("failed to extract config.json: %w", err)
	}

	// ensure atomic mutation of Lambdas map
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	s.Lambdas[functionName] = &LambdaEntry{
		Config: cfg,
		Lock:   &sync.Mutex{},
	}

	s.registerTriggers(functionName, cfg)

	return nil
}

func (s *LambdaStore) registerTriggers(functionName string, cfg *common.LambdaConfig) {
	s.events.Register(functionName, cfg.Triggers)
}

func (s *LambdaStore) unregisterTriggers(functionName string) {
	s.events.Unregister(functionName)
}

// assumes the caller holds the function lock
func (s *LambdaStore) addToRegistry(name string, body io.Reader) error {
	tarPath := filepath.Join(s.StorePath, name+".tar.gz")

	tarFile, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("failed to create lambda tarball: %w", err)
	}
	defer tarFile.Close()

	// TODO: atomically write tarball by writing to a temp file and renaming
	if _, err := io.Copy(tarFile, body); err != nil {
		return fmt.Errorf("failed to write tarball: %w", err)
	}

	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		// Clean up the bad tarball file
		_ = os.Remove(tarPath)
		return fmt.Errorf("failed to extract config: %w", err)
	}

	s.mapLock.Lock()

	entry, ok := s.Lambdas[name]
	if !ok {
		entry = &LambdaEntry{
			Lock: &sync.Mutex{},
		}
		s.Lambdas[name] = entry
	}
	entry.Config = cfg

	s.mapLock.Unlock()

	s.registerTriggers(name, cfg)

	return nil
}

// assumes the caller holds the function lock
func (s *LambdaStore) removeFromRegistry(name string) error {
	tarPath := filepath.Join(s.StorePath, name+".tar.gz")
	if err := os.Remove(tarPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove tarball for %s: %w", name, err)
	}

	s.mapLock.Lock()
	delete(s.Lambdas, name)
	s.mapLock.Unlock()

	s.unregisterTriggers(name)

	return nil
}

// getFuncLock safely returns the lock for a given function name
func (s *LambdaStore) getFuncLock(name string) *sync.Mutex {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	entry, ok := s.Lambdas[name]
	if !ok {
		entry = &LambdaEntry{
			Lock:   &sync.Mutex{},
			Config: nil,
		}
		s.Lambdas[name] = entry
	}
	return entry.Lock
}
