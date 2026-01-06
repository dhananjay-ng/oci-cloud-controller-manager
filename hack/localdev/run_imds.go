package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	component := "default"
	if len(os.Args) > 1 {
		component = os.Args[1]
	}

	var filePath string
	switch component {
	case "csi":
		filePath = "hack/localdev/imds_response_csi.json"
	case "ccm":
		filePath = "hack/localdev/imds_response_ccm.json"
	default:
		filePath = "hack/localdev/imds_response.json"
	}

	http.HandleFunc("/opc/v2/instance/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading file %s: %v", filePath, err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		var metadata map[string]interface{}
		if err := json.Unmarshal(data, &metadata); err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		log.Println("Response sent successfully")
	})

	log.Printf("Starting mock IMDS server at http://127.0.0.1:8081/opc/v2/instance/ using %s", filePath)
	if err := http.ListenAndServe("127.0.0.1:8081", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
