package cloudvm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"strconv"

	"github.com/open-lambda/open-lambda/ol/common"
)

func LoadWorkerConfigTemplate(path string, workerConfigPath string) error {
	// Ensure common.Conf is initialized
	if common.Conf == nil {
		if err := common.LoadConf(workerConfigPath); err != nil {
			return fmt.Errorf("failed to initialize common.Conf: %v", err)
		}
	}

	// Check if template.json exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		common.Conf.Worker_port = "6000" // TODO: read from boss config
		if err := common.SaveConf(workerConfigPath); err != nil {
			return fmt.Errorf("failed to save updated configuration: %v", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("error checking template.json: %v", err)
	}

	// Read template.json
	configRaw, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template.json: %v", err)
	}

	// Parse the template into a temporary Config struct
	var templateConfig common.Config
	if err := json.Unmarshal(configRaw, &templateConfig); err != nil {
		return fmt.Errorf("failed to parse template.json: %v", err)
	}

	// Use reflection to dynamically update common.Conf
	if err := updateConfig(common.Conf, &templateConfig); err != nil {
		return fmt.Errorf("failed to update configuration: %v", err)
	}

	// Save the updated configuration
	if common.Conf.Worker_dir == "" {
		return fmt.Errorf("Worker_dir is not set")
	}

	if err := common.SaveConf(workerConfigPath); err != nil {
		return fmt.Errorf("failed to save updated configuration: %v", err)
	}

	return nil
}

// updateConfig dynamically updates the target Config with values from the source Config
func updateConfig(target, source any) error {
	targetVal := reflect.ValueOf(target).Elem()
	sourceVal := reflect.ValueOf(source).Elem()

	for i := 0; i < targetVal.NumField(); i++ {
		targetField := targetVal.Field(i)
		sourceField := sourceVal.Field(i)

		// Skip unexported fields
		if !targetField.CanSet() {
			continue
		}

		// Handle nested structs recursively
		if targetField.Kind() == reflect.Struct {
			if err := updateConfig(targetField.Addr().Interface(), sourceField.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		// Update the target field if the source field is non-zero
		if !reflect.DeepEqual(sourceField.Interface(), reflect.Zero(sourceField.Type()).Interface()) {
			targetField.Set(sourceField)
		}
	}

	return nil
}

// isPortFree checks if a port (as a string) is available for use
func isPortFree(portStr string) (bool, error) {
	// Parse the port string to an integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false, fmt.Errorf("invalid port: %v", err)
	}

	// Check if the port is free
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil // Port is in use
	}
	listener.Close()
	return true, nil // Port is free
}
