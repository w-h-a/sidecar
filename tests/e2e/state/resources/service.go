package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const servicePort = 3000

const stateURL = "http://localhost:3501/state"

type RequestResponse struct {
	StartTime int            `json:"start_time,omitempty"`
	EndTime   int            `json:"end_time,omitempty"`
	States    []SidecarState `json:"states,omitempty"`
	Message   string         `json:"message,omitempty"`
}

type SidecarState struct {
	Key   string        `json:"key,omitempty"`
	Value *ServiceState `json:"value,omitempty"`
}

type ServiceState struct {
	Data string `json:"data,omitempty"`
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler has been called")

	w.WriteHeader(http.StatusOK)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("testHander has been called")

	var req RequestResponse

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RequestResponse{
			Message: err.Error(),
		})
		return
	}

	statusCode := http.StatusOK

	uri := r.URL.RequestURI()

	cmd := mux.Vars(r)["command"]

	rsp := RequestResponse{}

	var err error

	rsp.StartTime = int(time.Now().UnixMilli())

	switch cmd {
	case "create":
		err = create(req.States)
	case "list":
		rsp.States, err = list()
	case "delete":
		err = delete(req.States)
	default:
		err = fmt.Errorf("invalid URI: %s", uri)
		statusCode = http.StatusBadRequest
		rsp.Message = err.Error()
	}

	if err != nil && statusCode == http.StatusOK {
		statusCode = http.StatusInternalServerError
		rsp.Message = err.Error()
	}

	rsp.EndTime = int(time.Now().UnixMilli())

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(rsp)
}

func create(states []SidecarState) error {
	log.Printf("processing create request for %d entries", len(states))

	bs, err := json.Marshal(states)
	if err != nil {
		return err
	}

	rsp, err := http.Post(fmt.Sprintf("%s/%s", stateURL, "test"), "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return err
	}

	defer rsp.Body.Close()

	return nil
}

func list() ([]SidecarState, error) {
	log.Println("processing list request")

	rsp, err := http.Get(fmt.Sprintf("%s/%s", stateURL, "test"))
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	var states []SidecarState

	if err := json.Unmarshal(body, &states); err != nil {
		return nil, err
	}

	return states, nil
}

func delete(states []SidecarState) error {
	log.Printf("processing delete request for %d entries", len(states))

	for _, state := range states {
		url := fmt.Sprintf("%s/%s/%s", stateURL, "test", state.Key)

		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}

		client := &http.Client{}

		rsp, err := client.Do(req)
		if err != nil {
			return err
		}

		rsp.Body.Close()
	}

	return nil
}

func appRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/test/{command}", testHandler).Methods("POST")

	return router
}

func main() {
	log.Printf("state test service is listening on port %d", servicePort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", servicePort), appRouter()); err != nil {
		log.Fatal(err)
	}
}
