package main

import (
	"fmt"
	"log"
	"net/http"
)

const servicePort = 3000

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received request")
	w.Write([]byte("Hello, World"))
}

func main() {
	log.Printf("hello world test service is listening on port %d", servicePort)

	http.HandleFunc("/", rootHandler)

	http.ListenAndServe(fmt.Sprintf(":%d", servicePort), nil)
}
