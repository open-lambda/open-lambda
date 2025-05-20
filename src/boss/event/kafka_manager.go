package event

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

type KafkaManager struct {
	mapLock   sync.Mutex
	functions map[string][]common.KafkaTrigger // functions with active Kafka triggers
}

func NewKafkaManager() *KafkaManager {
	return &KafkaManager{
		functions: make(map[string][]common.KafkaTrigger),
	}
}

func (k *KafkaManager) Register(functionName string, triggers []common.KafkaTrigger) {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	// If already registered, tear down old config
	if _, exists := k.functions[functionName]; exists {
		k.Unregister(functionName)
	}

	// TODO: should we send a one big request with all the kafka trigger or send each trigger to different workers?

	// Setup each trigger at the worker
	for _, trigger := range triggers {
		// validate the required fields, make sure it is not empty
		if len(trigger.Topics) == 0 || len(trigger.Bootstrap_servers) == 0 {
			log.Printf("[KafkaManager] Skipping empty Kafka trigger for %s", functionName)
			continue
		}

		data, err := json.Marshal(trigger)
		if err != nil {
			log.Printf("[KafkaManager] Failed to marshal Kafka trigger for %s: %v", functionName, err)
			continue
		}

		setupURL := "http://localhost:5000/kafka-init/" + functionName // how to pick worker? Load balancing?
		resp, err := http.Post(setupURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("[KafkaManager] Failed to send setup request for %s: %v", functionName, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[KafkaManager] Worker returned error on setup for %s: %s", functionName, resp.Status)
		}
		resp.Body.Close()
	}

	// Track all triggers
	k.functions[functionName] = triggers
	log.Printf("[KafkaManager] Kafka consumer(s) registered for %s", functionName)
}

func (k *KafkaManager) Unregister(functionName string) {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	if _, exists := k.functions[functionName]; !exists {
		return
	}

	unsetupURL := "http://localhost:5000/kafka-stop/" + functionName // how to know which worker has the kafka consumer setup. save it in the map somehow?
	resp, err := http.Post(unsetupURL, "application/json", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		log.Printf("[KafkaManager] Failed to send unsetup request for %s: %v", functionName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[KafkaManager] Worker returned error on unsetup for %s: %s", functionName, resp.Status)
		return
	}

	delete(k.functions, functionName)
	log.Printf("[KafkaManager] Kafka consumer unregistered for %s", functionName)
}
