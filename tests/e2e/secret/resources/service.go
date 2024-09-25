package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const servicePort = 3000

const secretURL = "http://localhost:3501/secret"

type RequestResponse struct {
	StartTime int             `json:"start_time,omitempty"`
	EndTime   int             `json:"end_time,omitempty"`
	Secrets   []SidecarSecret `json:"secrets,omitempty"`
	Message   string          `json:"message,omitempty"`
}

type SidecarSecret struct {
	Store string            `json:"store,omitempty"`
	Key   string            `json:"key,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

func serviceRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/test/{command}", testHandler).Methods("POST")

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler has been called")

	w.WriteHeader(http.StatusOK)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("testHandler has been called")

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
	case "get":
		rsp.Secrets, err = getAll(req.Secrets)
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

func getAll(secrets []SidecarSecret) ([]SidecarSecret, error) {
	log.Printf("processing get request for %d secrets.", len(secrets))

	output := make([]SidecarSecret, 0, len(secrets))

	for _, secret := range secrets {
		data, err := get(secret.Store, secret.Key)
		if err != nil {
			return nil, err
		}

		output = append(output, SidecarSecret{
			Store: secret.Store,
			Key:   secret.Key,
			Data:  data,
		})
	}

	return output, nil
}

func get(store, key string) (map[string]string, error) {
	log.Printf("processing get request for store %s and key %s", store, key)

	url := fmt.Sprintf("%s/%s/%s", secretURL, store, key)

	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	secret := SidecarSecret{}

	if err := json.Unmarshal(body, &secret); err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func main() {
	log.Printf("state test service is listening on port %d", servicePort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", servicePort), serviceRouter()); err != nil {
		log.Fatal(err)
	}
}
