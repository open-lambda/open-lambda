package boss

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const LAMBDA_STORE_PATH = "./lambdaStore/" // Directory for storing lambdas

// Ensure lambdaStore directory exists at initialization
func init() {
	if err := os.MkdirAll(LAMBDA_STORE_PATH, 0755); err != nil {
		log.Fatalf("Failed to create lambda store directory: %v", err)
	}
}

// UploadLambda handles POST requests to upload a tar file to the lambda store
// URL Format: /lambda/upload/{function_name}
// Request Body: Contains the tar.gz file.
func UploadLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/lambda/upload/")
	log.Printf("Received request to upload lambda: %s\n", functionName)

	tarFilePath := filepath.Join(LAMBDA_STORE_PATH, functionName+".tar.gz")
	file, err := os.Create(tarFilePath)
	if err != nil {
		http.Error(w, "Failed to create tar file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Failed to save tar file", http.StatusInternalServerError)
		return
	}
	log.Printf("Lambda %s uploaded successfully!", functionName)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Lambda %s uploaded successfully", functionName)
}

// ListLambda lists all lambdas in the store
// URL: /lambda/list
// Lists all tar.gz files in the lambdaStore directory, returning the function names without the .tar.gz suffix.
func ListLambda(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(LAMBDA_STORE_PATH)
	if err != nil {
		http.Error(w, "Failed to list lambdas", http.StatusInternalServerError)
		return
	}

	var lambdaFunctions []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.gz") {
			lambdaFunctions = append(lambdaFunctions, strings.TrimSuffix(file.Name(), ".tar.gz"))
		}
	}

	json.NewEncoder(w).Encode(lambdaFunctions)
}

// DeleteLambda deletes a lambda function tar from the
// URL Format: /lambda/{function_name}
// Deletes the corresponding tar.gz file from the lambdaStore directory.
func DeleteLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/lambda/")
	filePath := filepath.Join(LAMBDA_STORE_PATH, functionName+".tar.gz")

	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Failed to delete lambda function", http.StatusInternalServerError)
		return
	}
	log.Printf("Lambda %s deleted successfully!", functionName)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s deleted successfully", functionName)
}

// UpdateLambda updates the tar file of an existing lambda
// URL Format: /lambda/update/{function_name}
// Request Body: Contains the new tar.gz file to overwrite the existing file with the same name.
func UpdateLambda(w http.ResponseWriter, r *http.Request) {
	functionName := strings.TrimPrefix(r.URL.Path, "/lambda/update/")
	log.Printf("Received request to update lambda: %s\n", functionName)

	tarFilePath := filepath.Join(LAMBDA_STORE_PATH, functionName+".tar.gz")
	file, err := os.Create(tarFilePath)
	if err != nil {
		http.Error(w, "Failed to create tar file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Failed to update tar file", http.StatusInternalServerError)
		return
	}
	log.Printf("Lambda %s updated successfully!", functionName)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Lambda %s updated successfully", functionName)
}
