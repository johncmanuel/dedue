package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/johncmanuel/dedue/internal/balancer"
	"github.com/johncmanuel/dedue/internal/strategy"
)

func main() {
	// parse command line flags
	listen := flag.String("listen", ":8080", "address to listen on")
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("usage: dedue [-listen :8080] <backend-url>...")
	}

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()

	// add backends from command line args
	for _, address := range flag.Args() {
		backend, err := url.Parse(address)
		if err != nil || (backend.Scheme != "http" && backend.Scheme != "https") || backend.Host == "" {
			log.Fatalf("invalid backend URL: %q", address)
		}
		// assume weight of 1 to keep things simple rn
		d.AddServer(address, 1)
	}

	go d.PeriodicHealthCheck(context.Background(), time.Minute)

	log.Printf("listening on %s with %d backends", *listen, flag.NArg())
	if err := http.ListenAndServe(*listen, d); err != nil {
		log.Fatal(err)
	}
}
