package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Machine struct {
	SystemID   string `json:"system_id"`
	StatusName string `json:"status_name"`
}

func main() {
	// Full API key as a single string
	fullApiKey := "dwXUe7WPvDeJCdj9wZ:YkpNHyMBtrWYgc54M6:9PJFbhnzuwYSGT2x5D6fEV6qsgLSpRye"
	apiKeyParts := strings.Split(fullApiKey, ":")

	maasURL := "http://192.168.200.3:5240/MAAS/api/2.0/machines/"

	// Generate oauth_nonce and oauth_timestamp
	nonce := uuid.New()
	timestamp := time.Now().Unix()

	// Prepare the OAuth 1.0 Authorization header
	authHeader := fmt.Sprintf(`OAuth oauth_version="1.0", oauth_signature_method="PLAINTEXT", oauth_consumer_key="%s", oauth_token="%s", oauth_signature="&%s", oauth_nonce="%s", oauth_timestamp="%d"`,
		apiKeyParts[0], apiKeyParts[1], apiKeyParts[2], nonce.String(), timestamp)

	req, err := http.NewRequest("GET", maasURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("Failed to fetch data: %s\n", resp.Status)
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	var machines []Machine
	err = json.Unmarshal(body, &machines)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	if len(machines) == 0 {
		fmt.Println("No machines found.")
		return
	}

	for _, machine := range machines {
		fmt.Printf("ID: %s, Status: %s\n", machine.SystemID, machine.StatusName)
	}
}

