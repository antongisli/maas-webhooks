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

	"maaswebhooks/maasmock"

	"github.com/google/uuid"
)

// individual machines with ability to lock
type Machine struct {
	SystemID   string `json:"system_id"`
	StatusName string `json:"status_name"`
}

// a slice of Machines with ability to lock
// TODO - move the mutex to the machine itself, and move the lock management to the
type MachineRegistry struct {
	Machines map[string]Machine
	mutex    sync.RWMutex
}

// this function allows updating the machine registry
func (r *MachineRegistry) UpdateMachine(machine Machine) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Check if the machine doesn't OR if the status has changed (either we are adding new machine or updating existing)
	if existingMachine, ok := r.Machines[machine.SystemID]; !ok || existingMachine.StatusName != machine.StatusName {
		r.Machines[machine.SystemID] = machine
		fmt.Printf("Updated/added machine %s with status %s\n", machine.SystemID, machine.StatusName)
		return true
	}
	return false
}

// get latest list from MAAS, compare to local registry and update it, and return a list of all updated machine IDs
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

	maasURL := "http://192.168.200.3:5240/MAAS/api/2.0/machines/"
	timer := 15

	testing := os.Getenv("TESTING")

	if testing == "true" {
		go maasmock.StartMockServer()
		maasURL = "http://localhost:5240/MAAS/api/2.0/machines/"
	}

	// prefer API keys from command line, but fallback to environment variable
	fullApiKey := flag.String("apikey", "", "API key for MAAS (overrides MAAS_API_KEY env var if set)")
	flag.Parse()
	if fullApiKey == nil {
		fullApiKey2 := os.Getenv("MAAS_API_KEY")
		fullApiKey = &fullApiKey2
	}

	// Check if the API key is still empty
	if *fullApiKey == "" {
		fmt.Println("Error: API key is not provided. Set MAAS_API_KEY environment variable or use the -apikey flag.")
		os.Exit(1) // Exit with an error code
	}

	// TODO there is a thing called a syncmap in Go
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
		_, updatedMachines := streamAndUpdateMachines(&registry, *fullApiKey, maasURL)
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
