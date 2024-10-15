package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
)

const (
	servicePort = 3000
)

var receivedMessageDummy []map[string]interface{}

var mtx sync.RWMutex

type ServiceResponse struct {
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ReceivedMessagesResponse struct {
	ReceivedByTopicDummy []map[string]interface{} `json:"dummy-topic"`
}

func serviceRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/go/dummy", dummyHandler).Methods("POST")
	router.HandleFunc("/test", testHandler).Methods("POST")

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler has been called")

	w.WriteHeader(http.StatusOK)
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("dummyHandler has been called")

	defer r.Body.Close()

	var err error

	var data []byte

	var body []byte

	if r.Body != nil {
		if data, err = io.ReadAll(r.Body); err == nil {
			body = data
		}
	} else {
		err = errors.New("r.Body is nil")
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ServiceResponse{
			Message: err.Error(),
		})
		return
	}

	msg, err := extractMessage(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ServiceResponse{
			Message: err.Error(),
		})
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	receivedMessageDummy = append(receivedMessageDummy, msg)

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(ServiceResponse{
		Message: "Success",
	})
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("testHandler has been called")

	mtx.RLock()
	defer mtx.RUnlock()

	rsp := ReceivedMessagesResponse{
		ReceivedByTopicDummy: receivedMessageDummy,
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(rsp)
}

func extractMessage(body []byte) (map[string]interface{}, error) {
	log.Printf("extractMessage has been called")

	log.Printf("body: %s", string(body))

	payload := sidecar.Payload{}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	m := map[string]interface{}{}

	if err := json.Unmarshal(payload.Data, &m); err != nil {
		return nil, err
	}

	log.Printf("output: '%+v'\n", m)

	return m, nil
}

func main() {
	log.Printf("subscribe test service is listening on port %d", servicePort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", servicePort), serviceRouter()); err != nil {
		log.Fatal(err)
	}
}
