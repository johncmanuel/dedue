package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/johncmanuel/dedue/internal/balancer"
	"github.com/johncmanuel/dedue/internal/strategy"
)

func main() {
	// parse command line flags
	listen := flag.String("listen", ":8080", "address to listen on")
	strategyName := flag.String("strategy", "round-robin", "balancing strategy: round-robin or random")
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("usage: dedue [-listen :8080] [-strategy round-robin|random] <backend-url[,weight]>...")
	}

	d := balancer.NewDedue()

	switch *strategyName {
	case "round-robin":
		d.Strategy = strategy.NewRoundRobin()
	case "random":
		d.Strategy = strategy.NewRandom()
	default:
		log.Fatalf("unknown strategy %q: choose round-robin or random", *strategyName)
	}

	// add backends from command line args
	for _, arg := range flag.Args() {
		address, weight, err := parseBackendArg(arg)
		if err != nil {
			log.Fatal(err)
		}
		d.AddServer(address, weight)
	}

	go d.PeriodicHealthCheck(context.Background(), time.Minute)

	log.Printf("listening on %s with %d backends using %s", *listen, flag.NArg(), *strategyName)
	if err := http.ListenAndServe(*listen, d); err != nil {
		log.Fatal(err)
	}
}

func parseBackendArg(arg string) (string, int, error) {
	address, weightText, hasWeight := strings.Cut(arg, ",")
	weight := 1
	if hasWeight {
		parsedWeight, err := strconv.Atoi(weightText)
		if err != nil {
			return "", 0, fmt.Errorf("invalid backend weight %q: %w", weightText, err)
		}
		if parsedWeight <= 0 {
			return "", 0, fmt.Errorf("backend weight must be positive: %d", parsedWeight)
		}
		weight = parsedWeight
	}

	backend, err := url.Parse(address)
	if err != nil {
		return "", 0, fmt.Errorf("invalid backend URL %q: %w", address, err)
	}
	if (backend.Scheme != "http" && backend.Scheme != "https") || backend.Host == "" {
		return "", 0, fmt.Errorf("invalid backend URL %q", address)
	}

	return address, weight, nil
}
