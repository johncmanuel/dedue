package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: mockbackend <port> <backend-id>")
	}

	id := os.Args[2]
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Backend", id)
		if _, err := fmt.Fprintln(w, id); err != nil {
			log.Printf("write response: %v", err)
		}
	})

	address := net.JoinHostPort("127.0.0.1", os.Args[1])
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
