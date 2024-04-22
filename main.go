package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// individual machines with ability to lock
type Machine struct {
	SystemID   string `json:"system_id"`
	StatusName string `json:"status_name"`
}

// a slice of Machines with ability to lock
type MachineRegistry struct {
	Machines map[string]Machine
	mutex    sync.RWMutex
}

func (r *MachineRegistry) UpdateMachine(machine Machine) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Check if the machine exists and if the status has changed
	if existingMachine, ok := r.Machines[machine.SystemID]; !ok || existingMachine.StatusName != machine.StatusName {
		r.Machines[machine.SystemID] = machine
		fmt.Printf("Updated machine %s to status %s\n", machine.SystemID, machine.StatusName)
		return true
	} else {
		return false
	}
}

func streamAndUpdateMachines(registry *MachineRegistry, fullApiKey string, maasURL string) (error, []Machine) {

	var updatedMachines []Machine
	apiKeyParts := strings.Split(fullApiKey, ":")

	// Generate oauth_nonce and oauth_timestamp
	nonce := uuid.New()
	timestamp := time.Now().Unix()

	// Prepare the OAuth 1.0 Authorization header
	authHeader := fmt.Sprintf(`OAuth oauth_version="1.0", oauth_signature_method="PLAINTEXT", oauth_consumer_key="%s", oauth_token="%s", oauth_signature="&%s", oauth_nonce="%s", oauth_timestamp="%d"`,
		apiKeyParts[0], apiKeyParts[1], apiKeyParts[2], nonce.String(), timestamp)

	// make request
	req, err := http.NewRequest("GET", maasURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err, updatedMachines
	}
	req.Header.Set("Authorization", authHeader)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err), updatedMachines
	}
	defer resp.Body.Close()

	// Check for HTTP error status
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch data: %s", resp.Status), updatedMachines
	}

	// Decode the stream
	decoder := json.NewDecoder(resp.Body)
	_, err = decoder.Token() // Read and discard the array start token
	if err != nil {
		return fmt.Errorf("error reading JSON stream start: %v", err), updatedMachines
	}

	for decoder.More() {
		var machine Machine
		if err := decoder.Decode(&machine); err != nil {
			return fmt.Errorf("error decoding machine: %v", err), updatedMachines
		}
		if registry.UpdateMachine(machine) {
			updatedMachines = append(updatedMachines, machine)
		}
	}

	_, err = decoder.Token() // Read and discard the array end token
	if err != nil {
		return fmt.Errorf("error reading JSON stream end: %v", err), updatedMachines
	}

	return nil, updatedMachines
}

func main() {

	// Define a command-line flag for the API key
	fullApiKey := flag.String("apikey", "", "API key for MAAS (overrides MAAS_API_KEY env var if set)")
	flag.Parse()

	// First, try to get the API key from the environment variable
	if *fullApiKey == "" {
		*fullApiKey = os.Getenv("MAAS_API_KEY")
	}

	// Check if the API key is still empty
	if *fullApiKey == "" {
		fmt.Println("Error: API key is not provided. Set MAAS_API_KEY environment variable or use the -apikey flag.")
		os.Exit(1) // Exit with an error code
	}

	maasURL := "http://192.168.200.3:5240/MAAS/api/2.0/machines/"
	timer := 15

	var registry = MachineRegistry{
		Machines: make(map[string]Machine),
		mutex:    sync.RWMutex{},
	}

	registry.UpdateMachine(Machine{SystemID: "TEST", StatusName: "TESTSTATUS1"})

	// initial run populates machine registry
	fmt.Println("Initializing machine registry ...\n")
	_, updatedMachines := streamAndUpdateMachines(&registry, *fullApiKey, maasURL)

	for _, machine := range updatedMachines {
		fmt.Printf("%s\n", machine.SystemID)
	}
	fmt.Println("Finished.\nStarting state detection loop:\n")

	// start looping
	for {
		time.Sleep(time.Duration(timer) * time.Second)
		fmt.Println("Starting changed state check.\n")
		_, updatedMachines := streamAndUpdateMachines(&registry, fullApiKey, maasURL)
		fmt.Println("Machines with changed state:\n")
		for _, machine := range updatedMachines {
			fmt.Printf("%s\n", machine.SystemID)
		}
	}

	//registry.UpdateMachine(Machine{SystemID: "TEST", StatusName: "TESTSTATUS2"})

	for _, machine := range updatedMachines {
		fmt.Printf("%s\n", machine.SystemID)
	}

}
