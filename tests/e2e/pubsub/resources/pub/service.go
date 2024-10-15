package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
)

const (
	servicePort = 3000
	publishUrl  = "http://localhost:3501/publish"
)

type PublishCommand struct {
	Topic string                 `json:"topic"`
	Data  map[string]interface{} `json:"data"`
}

type ServiceResponse struct {
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
	Message   string `json:"message,omitempty"`
}

func serviceRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/test", testHandler).Methods("POST")

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler has been called")

	w.WriteHeader(http.StatusOK)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("testHandler has been called")

	var commandBody PublishCommand

	if err := json.NewDecoder(r.Body).Decode(&commandBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ServiceResponse{
			Message: err.Error(),
		})
		return
	}

	rsp := ServiceResponse{}

	bs, _ := json.Marshal(commandBody.Data)

	event := sidecar.Event{
		EventName: commandBody.Topic,
		Payload: sidecar.Payload{
			Metadata: map[string]string{},
			Data:     bs,
		},
	}

	rsp.StartTime = int(time.Now().UnixMilli())

	bs, err := json.Marshal(event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ServiceResponse{
			Message: err.Error(),
		})
		return
	}

	sidecarRsp, err := http.Post(publishUrl, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		log.Printf("publish failed: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ServiceResponse{
			Message: err.Error(),
		})
		return
	}

	defer sidecarRsp.Body.Close()

	w.WriteHeader(http.StatusOK)

	rsp.Message = "Success"

	rsp.EndTime = int(time.Now().UnixMilli())

	json.NewEncoder(w).Encode(rsp)
}

func main() {
	log.Printf("publish test service is listening on port %d", servicePort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", servicePort), serviceRouter()); err != nil {
		log.Fatal(err)
	}
}
