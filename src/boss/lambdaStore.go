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

const LAMBDA_STORE_PATH = "./lambdaStore/"

var (
	lambdaRegistry      []LambdaEntry
	httpTriggerRegistry []HTTPEntry
	cronTriggerRegistry []CronEntry
)

type LambdaEntry struct {
	FunctionName string
	Config       *common.LambdaConfig
}

type HTTPEntry struct {
	FunctionName string
	Method       string
}

type CronEntry struct {
	FunctionName string
	Schedule     string
}

func init() {
	info, err := os.Stat(LAMBDA_STORE_PATH)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(LAMBDA_STORE_PATH, 0755); err != nil {
			log.Fatalf("Failed to create lambda store directory: %v", err)
		}
	} else if err == nil && info.IsDir() {
		files, _ := os.ReadDir(LAMBDA_STORE_PATH)
		for _, file := range files {
			if file.IsDir() {
				funcName := file.Name()
				if err := loadConfigAndRegister(funcName); err != nil {
					log.Printf("Failed to load existing lambda %s: %v", funcName, err)
				}
			}
		}
	} else if err != nil {
		log.Fatalf("Error checking lambda store directory: %v", err)
	}
}

func UploadLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/lambda/upload/")
	destDir := filepath.Join(LAMBDA_STORE_PATH, functionName)

	// Clean any existing code
	os.RemoveAll(destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		http.Error(w, "Failed to create lambda directory", http.StatusInternalServerError)
		return
	}

	// Temporarily save uploaded tar.gz
	tempTarPath := filepath.Join(os.TempDir(), functionName+".tar.gz")
	tempFile, err := os.Create(tempTarPath)
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

	// Extract into lambdaStore/{functionName}/
	if err := ExtractTarGz(tempTarPath, destDir); err != nil {
		http.Error(w, "Failed to extract lambda archive", http.StatusInternalServerError)
		os.Remove(tempTarPath)
		return
	}
	os.Remove(tempTarPath) // Clean up temp file

	// Register the lambda config
	if err := loadConfigAndRegister(functionName); err != nil {
		log.Printf("Warning: Lambda %s uploaded but failed to register config: %v", functionName, err)
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded and registered successfully", functionName)
}

func ListLambda(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(LAMBDA_STORE_PATH)
	if err != nil {
		http.Error(w, "Failed to list lambdas", http.StatusInternalServerError)
		return
	}

	var lambdaFunctions []string
	for _, file := range files {
		if file.IsDir() {
			lambdaFunctions = append(lambdaFunctions, file.Name())
		}
	}

	json.NewEncoder(w).Encode(lambdaFunctions)
}

func DeleteLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/lambda/")
	dirPath := filepath.Join(LAMBDA_STORE_PATH, functionName)

	if err := os.RemoveAll(dirPath); err != nil {
		http.Error(w, "Failed to delete lambda function", http.StatusInternalServerError)
		return
	}
	unregisterTriggers(functionName)
	removeFromRegistry(functionName)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", functionName)
}

func UpdateLambda(w http.ResponseWriter, r *http.Request) {
	UploadLambda(w, r) // should we keep this at all? or just do update thru upload?
}

func ListHTTPTriggers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(httpTriggerRegistry)
}

func ListCronTriggers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(cronTriggerRegistry)
}

func loadConfigAndRegister(functionName string) error {
	destDir := filepath.Join(LAMBDA_STORE_PATH, functionName)

	lambdaConfig, err := common.LoadLambdaConfig(destDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	unregisterTriggers(functionName)
	removeFromRegistry(functionName)

	addToRegistry(functionName, lambdaConfig)
	registerTriggers(functionName, lambdaConfig)

	return nil
}

func ExtractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func registerTriggers(functionName string, cfg *common.LambdaConfig) {
	for _, trigger := range cfg.Triggers.HTTP {
		httpTriggerRegistry = append(httpTriggerRegistry, HTTPEntry{
			FunctionName: functionName,
			Method:       strings.ToUpper(trigger.Method),
		})
	}

	for _, trigger := range cfg.Triggers.Cron {
		cronTriggerRegistry = append(cronTriggerRegistry, CronEntry{
			FunctionName: functionName,
			Schedule:     trigger.Schedule,
		})
	}
}

func unregisterTriggers(functionName string) {
	newHTTP := []HTTPEntry{}
	for _, entry := range httpTriggerRegistry {
		if entry.FunctionName != functionName {
			newHTTP = append(newHTTP, entry)
		}
	}
	httpTriggerRegistry = newHTTP

	newCronList := []CronEntry{}
	for _, entry := range cronTriggerRegistry {
		if entry.FunctionName != functionName {
			newCronList = append(newCronList, entry)
		}
	}
	cronTriggerRegistry = newCronList
}

func addToRegistry(functionName string, config *common.LambdaConfig) {
	lambdaRegistry = append(lambdaRegistry, LambdaEntry{
		FunctionName: functionName,
		Config:       config,
	})
}

func removeFromRegistry(functionName string) {
	newRegistry := []LambdaEntry{}
	for _, entry := range lambdaRegistry {
		if entry.FunctionName != functionName {
			newRegistry = append(newRegistry, entry)
		}
	}
	lambdaRegistry = newRegistry
}
