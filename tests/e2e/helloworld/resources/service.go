package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const servicePort = 3000

type AppResponse struct {
	Message   string `json:"message,omitempty"`
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
}

type TestCommandRequest struct {
	Message string `json:"message,omitempty"`
}

func serviceRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/tests/{test}", testHandler).Methods("POST")

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler has been called")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AppResponse{Message: "OK"})
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("testHander has been called")

	testCommand := mux.Vars(r)["test"]

	var commandBody TestCommandRequest

	if err := json.NewDecoder(r.Body).Decode(&commandBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AppResponse{Message: err.Error()})
		return
	}

	rsp := AppResponse{Message: fmt.Sprintf("%s is not supported", testCommand)}

	statusCode := http.StatusBadRequest

	rsp.StartTime = int(time.Now().UnixMilli())

	switch testCommand {
	case "blue":
		statusCode, rsp.Message = blue(commandBody)
	case "green":
		statusCode, rsp.Message = green(commandBody)
	}

	rsp.EndTime = int(time.Now().UnixMilli())

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(rsp)
}

func blue(commandRequest TestCommandRequest) (int, string) {
	log.Printf("blue test message: %s", commandRequest.Message)
	return http.StatusOK, "Hello, Blue"
}

func green(commandRequest TestCommandRequest) (int, string) {
	log.Printf("green test message: %s", commandRequest.Message)
	return http.StatusOK, "Hello, Green"
}

func main() {
	log.Printf("hello world test service is listening on port %d", servicePort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", servicePort), serviceRouter()); err != nil {
		log.Fatal(err)
	}
}
