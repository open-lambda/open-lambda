// boss/worker_config.go
package cloudvm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/open-lambda/open-lambda/ol/common"
)

func LoadWorkerConfigTemplate(path string) error {
	// Check if template.json exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Do nothing if file does not exist
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
	configPath := filepath.Join(common.Conf.Worker_dir, "config.json")
	if err := common.SaveConf(configPath); err != nil {
		return fmt.Errorf("failed to save updated configuration: %v", err)
	}

	return nil
}

// updateConfig dynamically updates the target Config with values from the source Config
func updateConfig(target, source *common.Config) error {
	targetVal := reflect.ValueOf(target).Elem()
	sourceVal := reflect.ValueOf(source).Elem()

	for i := 0; i < targetVal.NumField(); i++ {
		targetField := targetVal.Field(i)
		sourceField := sourceVal.Field(i)

		// Skip unexported fields
		if !targetField.CanSet() {
			continue
		}

		// Update the target field if the source field is non-zero
		if !reflect.DeepEqual(sourceField.Interface(), reflect.Zero(sourceField.Type()).Interface()) {
			targetField.Set(sourceField)
		}
	}

	return nil
}
