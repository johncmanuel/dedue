package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	healthDone := make(chan struct{})
	go func() {
		defer close(healthDone)
		d.PeriodicHealthCheck(ctx, 5*time.Second)
	}()

	server := &http.Server{Addr: *listen, Handler: d}
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s with %d backends", *listen, flag.NArg())
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		stop()
		<-healthDone
		log.Fatal(err)
	}
	if err := <-shutdownDone; err != nil {
		log.Printf("shut down server: %v", err)
	}
	<-healthDone
}
