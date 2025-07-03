package event

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
)

// UploadLambda handles POST /registry/{name} - saves to common.Conf.Registry
func UploadLambda(w http.ResponseWriter, r *http.Request) {
	funcName := strings.TrimPrefix(r.URL.Path, REGISTRY_BASE_PATH)

	if err := common.ValidateFunctionName(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure registry directory exists
	if err := os.MkdirAll(common.Conf.Registry, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create registry directory: %v", err), http.StatusInternalServerError)
		return
	}

	tarPath := filepath.Join(common.Conf.Registry, funcName+".tar.gz")
	tmpPath := tarPath + ".tmp"

	tarFile, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp tarball: %v", err), http.StatusInternalServerError)
		return
	}

	// Always try to clean up temp file on return
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tarFile, r.Body); err != nil {
		tarFile.Close()
		http.Error(w, fmt.Sprintf("Failed to write to temp tarball: %v", err), http.StatusInternalServerError)
		return
	}

	if err := tarFile.Close(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to close temp tarball: %v", err), http.StatusInternalServerError)
		return
	}

	// Validate the tarball by extracting config
	if _, err := common.ExtractConfigFromTarGz(tmpPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract config from tarball: %v", err), http.StatusBadRequest)
		return
	}

	// Atomically replace the old file
	if err := os.Rename(tmpPath, tarPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to rename temp tarball: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded successfully", funcName)
}

// DeleteLambda handles DELETE /registry/{name}
func DeleteLambda(w http.ResponseWriter, r *http.Request) {
	funcName := strings.TrimPrefix(r.URL.Path, REGISTRY_BASE_PATH)

	if err := common.ValidateFunctionName(funcName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tarPath := filepath.Join(common.Conf.Registry, funcName+".tar.gz")

	if err := os.Remove(tarPath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Lambda %s not found", funcName), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete lambda %s: %v", funcName, err), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", funcName)
}

// ListLambdas handles GET /registry
func ListLambdas(w http.ResponseWriter) {
	if _, err := os.Stat(common.Conf.Registry); os.IsNotExist(err) {
		// Registry directory doesn't exist, return empty list
		if err := json.NewEncoder(w).Encode([]string{}); err != nil {
			http.Error(w, "failed to encode lambda list", http.StatusInternalServerError)
		}
		return
	}

	files, err := os.ReadDir(common.Conf.Registry)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read registry directory: %v", err), http.StatusInternalServerError)
		return
	}

	funcNames := make([]string, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.gz") {
			funcName := strings.TrimSuffix(file.Name(), ".tar.gz")
			funcNames = append(funcNames, funcName)
		}
	}

	if err := json.NewEncoder(w).Encode(funcNames); err != nil {
		http.Error(w, "failed to encode lambda list", http.StatusInternalServerError)
	}
}

// RetrieveLambdaConfig handles GET /registry/{name}/config
func RetrieveLambdaConfig(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, REGISTRY_BASE_PATH)
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

	tarPath := filepath.Join(common.Conf.Registry, funcName+".tar.gz")
	cfg, err := common.ExtractConfigFromTarGz(tarPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Lambda %q not found", funcName), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to extract config: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, "failed to encode config as JSON", http.StatusInternalServerError)
		return
	}
}