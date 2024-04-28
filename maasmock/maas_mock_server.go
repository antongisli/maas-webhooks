package maasmock

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Machine struct {
	SystemID   string `json:"system_id"`
	StatusName string `json:"status_name"`
}

var machines = []Machine{
	{SystemID: "rfykrh", StatusName: "Broken"},
	{SystemID: "a3babp", StatusName: "Deployed"},
	{SystemID: "hyhypg", StatusName: "Deployed"},
	{SystemID: "tekmyk", StatusName: "Ready"},
	{SystemID: "a47eh3", StatusName: "Ready"},
	{SystemID: "bssbfh", StatusName: "Ready"},
	{SystemID: "4dneak", StatusName: "Ready"},
	{SystemID: "mbknk6", StatusName: "Deployed"},
	{SystemID: "pcfenf", StatusName: "Ready"},
	{SystemID: "s7468h", StatusName: "Deployed"},
	{SystemID: "7cpkgs", StatusName: "Ready"},
	{SystemID: "gm6tae", StatusName: "Ready"},
	{SystemID: "b8n4mg", StatusName: "Ready"},
	{SystemID: "xswsfr", StatusName: "Releasing failed"},
}

var states = []string{"Broken", "Deployed", "Ready", "Releasing failed"}

var mutex = &sync.Mutex{}

func StartMockServer() {
	http.HandleFunc("/MAAS/api/2.0/machines/", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		js, err := json.Marshal(machines)
		mutex.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})

	// Simulate machines transitioning state over time
	go func() {
		for {
			mutex.Lock()
			for i := 0; i < rand.Intn(4)+1; i++ {
				machineIndex := rand.Intn(len(machines))
				machines[machineIndex].StatusName = states[rand.Intn(len(states))]
			}
			mutex.Unlock()
			time.Sleep(10 * time.Second)
		}
	}()

	http.ListenAndServe(":5240", nil)
}

func main() {
	StartMockServer()
}
