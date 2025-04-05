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

	destDir := filepath.Join(s.StorePath, functionName)
	os.RemoveAll(destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		http.Error(w, "Failed to create lambda directory", http.StatusInternalServerError)
		return
	}

	tempPath := filepath.Join(os.TempDir(), functionName+".tar.gz")
	tempFile, err := os.Create(tempPath)
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(tempFile, r.Body); err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		tempFile.Close()
		return
	}
	tempFile.Close()

	if err := ExtractTarGz(tempPath, destDir); err != nil {
		http.Error(w, "Failed to extract archive", http.StatusInternalServerError)
		os.Remove(tempPath)
		return
	}
	os.Remove(tempPath)

	if err := s.loadConfigAndRegister(functionName); err != nil {
		log.Printf("Lambda %s uploaded but failed to register: %v", functionName, err)
	}

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

// ------------------- Core Logic ----------------------

func (s *LambdaStore) loadConfigAndRegister(functionName string) error {
	path := filepath.Join(s.StorePath, functionName)
	cfg, err := common.LoadLambdaConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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
	newList := []LambdaEntry{}
	for _, entry := range s.Lambdas {
		if entry.FunctionName != name {
			newList = append(newList, entry)
		}
	}
	s.Lambdas = newList
}

// ------------------- Utils ----------------------

func ExtractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("invalid .gz file: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid tar: %w", err)
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			log.Printf("Skipping unknown tar entry: %s", header.Name)
		}
	}
	return nil
}

func sanitizeFunctionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid function name")
	}
	return name, nil
}
