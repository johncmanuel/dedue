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
	strategyName := flag.String("strategy", "round-robin", "balancing strategy: round-robin, random, or weighted-round-robin")
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("usage: dedue [-listen :8080] [-strategy round-robin|random|weighted-round-robin] <backend-url[,weight]>...")
	}

	d := balancer.NewDedue()

	s, err := getStrategyFromFlag(*strategyName)
	if err != nil {
		log.Fatal(err)
	}
	d.Strategy = s

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

// parseBackendArg parses a backend argument in the form of "url[,weight]" and returns the address and weight.
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

// getStrategyFromFlag returns a BalancingStrategy based on the provided strategy name.
func getStrategyFromFlag(strategyName string) (strategy.BalancingStrategy, error) {
	switch strategyName {
	case "round-robin":
		return strategy.NewRoundRobin(), nil
	case "random":
		return strategy.NewRandom(), nil
	case "weighted-round-robin":
		return strategy.NewWeightedRoundRobin(), nil
	default:
		return nil, fmt.Errorf("unknown strategy %q: choose round-robin, random, or weighted-round-robin", strategyName)
	}
}
