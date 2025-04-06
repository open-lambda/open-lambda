package boss

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
)

type LambdaStore struct {
	StorePath    string
	Lambdas      []LambdaEntry
	HTTPTriggers []HTTPEntry
	CronTriggers []CronEntry
}

type LambdaEntry struct {
	FunctionName string
	Config       *common.LambdaConfig
}

type HTTPEntry struct {
	FunctionName string
	common.HTTPTrigger
}

type CronEntry struct {
	FunctionName string
	common.CronTrigger
}

func NewLambdaStore(storePath string) (*LambdaStore, error) {
	store := &LambdaStore{StorePath: storePath}

	if err := os.MkdirAll(store.StorePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lambda store directory: %w", err)
	}

	files, _ := os.ReadDir(store.StorePath)
	for _, file := range files {
		if file.IsDir() {
			funcName := file.Name()
			if err := store.loadConfigAndRegister(funcName); err != nil {
				log.Printf("Failed to load lambda %s: %v", funcName, err)
			}
		}
	}
	return store, nil
}

// ------------------- HTTP Handlers ----------------------

func (s *LambdaStore) UploadLambda(w http.ResponseWriter, r *http.Request) {
	rawName := strings.TrimPrefix(r.URL.Path, "/lambda/upload/")
	functionName, err := sanitizeFunctionName(rawName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save tar.gz to permanent store
	lambdaDir := filepath.Join(s.StorePath, functionName)
	os.RemoveAll(lambdaDir)
	if err := os.MkdirAll(lambdaDir, 0755); err != nil {
		http.Error(w, "Failed to create lambda directory", http.StatusInternalServerError)
		return
	}

	tarPath := filepath.Join(lambdaDir, functionName+".tar.gz")
	tarFile, err := os.Create(tarPath)
	if err != nil {
		http.Error(w, "Failed to create lambda file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(tarFile, r.Body); err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		tarFile.Close()
		return
	}
	tarFile.Close()

	// Reopen tarball to extract just config.json
	f, err := os.Open(tarPath)
	if err != nil {
		http.Error(w, "Failed to reopen uploaded tarball", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	cfg, err := extractConfigFromTarGz(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract config.json: %v", err), http.StatusBadRequest)
		return
	}

	s.unregisterTriggers(functionName)
	s.removeFromRegistry(functionName)
	s.addToRegistry(functionName, cfg)
	s.registerTriggers(functionName, cfg)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded successfully", functionName)
}

func (s *LambdaStore) DeleteLambda(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/lambda/")
	functionName, err := sanitizeFunctionName(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dir := filepath.Join(s.StorePath, functionName)
	if err := os.RemoveAll(dir); err != nil {
		http.Error(w, "Failed to delete lambda", http.StatusInternalServerError)
		return
	}

	s.unregisterTriggers(functionName)
	s.removeFromRegistry(functionName)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", functionName)
}

func (s *LambdaStore) ListLambda(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(s.StorePath)
	if err != nil {
		http.Error(w, "Failed to list lambdas", http.StatusInternalServerError)
		return
	}

	var names []string
	for _, file := range files {
		if file.IsDir() {
			names = append(names, file.Name())
		}
	}
	json.NewEncoder(w).Encode(names)
}

func (s *LambdaStore) GetLambdaConfig(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/lambda/config/")
	functionName, err := sanitizeFunctionName(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tarPath := filepath.Join(s.StorePath, functionName, "lambda.tar.gz")
	f, err := os.Open(tarPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open tarball: %v", err), http.StatusNotFound)
		return
	}
	defer f.Close()

	cfg, err := extractConfigFromTarGz(f)
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
	tarPath := filepath.Join(s.StorePath, functionName, "lambda.tar.gz")

	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open lambda tarball: %w", err)
	}
	defer f.Close()

	cfg, err := extractConfigFromTarGz(f)
	if err != nil {
		return fmt.Errorf("failed to extract config.json: %w", err)
	}

	s.unregisterTriggers(functionName)
	s.removeFromRegistry(functionName)
	s.addToRegistry(functionName, cfg)
	s.registerTriggers(functionName, cfg)

	return nil
}

func (s *LambdaStore) registerTriggers(functionName string, cfg *common.LambdaConfig) {
	for _, t := range cfg.Triggers.HTTP {
		s.HTTPTriggers = append(s.HTTPTriggers, HTTPEntry{
			FunctionName: functionName,
			HTTPTrigger:  t,
		})
	}
	for _, t := range cfg.Triggers.Cron {
		s.CronTriggers = append(s.CronTriggers, CronEntry{
			FunctionName: functionName,
			CronTrigger:  t,
		})
	}
}

func (s *LambdaStore) unregisterTriggers(functionName string) {
	var httpFiltered []HTTPEntry
	for _, entry := range s.HTTPTriggers {
		if entry.FunctionName != functionName {
			httpFiltered = append(httpFiltered, entry)
		}
	}
	s.HTTPTriggers = httpFiltered

	var cronFiltered []CronEntry
	for _, entry := range s.CronTriggers {
		if entry.FunctionName != functionName {
			cronFiltered = append(cronFiltered, entry)
		}
	}
	s.CronTriggers = cronFiltered
}

func (s *LambdaStore) addToRegistry(name string, cfg *common.LambdaConfig) {
	s.Lambdas = append(s.Lambdas, LambdaEntry{
		FunctionName: name,
		Config:       cfg,
	})
}

func (s *LambdaStore) removeFromRegistry(name string) {
	var newList []LambdaEntry
	for _, entry := range s.Lambdas {
		if entry.FunctionName != name {
			newList = append(newList, entry)
		}
	}
	s.Lambdas = newList
}

// ------------------- Utils ----------------------

func extractConfigFromTarGz(r io.Reader) (*common.LambdaConfig, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("invalid .gz file: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Create a temp dir to extract ol.yaml
	tempDir, err := os.MkdirTemp("", "lambda-config-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // clean up

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tar: %w", err)
		}

		if filepath.Base(header.Name) == "ol.yaml" {
			// Save ol.yaml to temp dir
			outPath := filepath.Join(tempDir, "ol.yaml")
			outFile, err := os.Create(outPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create temp ol.yaml: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return nil, fmt.Errorf("failed to write ol.yaml: %w", err)
			}
			outFile.Close()

			return common.LoadLambdaConfig(tempDir)
		}
	}

	return nil, fmt.Errorf("ol.yaml not found in archive")
}

func sanitizeFunctionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid function name")
	}
	return name, nil
}
