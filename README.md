# dedue

dedue is a load balancer written in Go. Created for learning purposes!

## Run

Start local mock backends and the load balancer:
```bash
./run_cluster.sh

# visit the load balancer
curl -i http://127.0.0.1:8080/
```

The Go mock backends return their ID in the body and `X-Backend` header. Each also serves `GET /health` with a 200 status. The balancer checks backend health at startup and every minute afterward. Press Ctrl-C in the cluster terminal to stop all servers. Use `./run_cluster.sh 3 9001 127.0.0.1:8080` to choose the backend count, first backend port, and listen address.

The default strategy is `round-robin`. To use `random`, run `./run_cluster.sh 3 9001 127.0.0.1:8080 random`, or pass `-strategy=random` to `go run .` before the backend URLs. Other strategy names are rejected.

Pass backend URLs directly to `go run .` to configure a static server list. Add `,weight` to set a positive weight for a backend; plain URLs default to weight 1:

```bash
go run . http://127.0.0.1:9001,3 http://127.0.0.1:9002,1
```

The current `round-robin` and `random` strategies do not use weights.

## Real time load testing

Install [Vegeta](https://github.com/tsenart/vegeta), then start the cluster with `./run_cluster.sh`. As an example, spin up a second terminal and send 100 requests per second for 30 seconds:

```bash
printf 'GET http://127.0.0.1:8080/\n' | vegeta attack -rate=100 -duration=30s | vegeta report
```

Repeat with higher `-rate` values (for example, 500 and 1000). Compare the achieved request rate, errors, and latency percentiles as load increases. To estimate the balancer's overhead, run the same command against a mock backend directly by replacing port `8080` with `9001` (the default first backend port). Keep the rate and duration the same for both runs.

For a repeatable run, keep `./run_cluster.sh` running and, in another terminal, run:

```bash
./scripts/load_test.sh                                    # 100, 500, and 1000 requests/s for 30s each
./scripts/load_test.sh http://127.0.0.1:8080/ 5s 20 100   # shorter custom run
./scripts/load_test.sh http://127.0.0.1:8080/ 5s 0        # unlimited rate with 20 workers
```

The script requires Vegeta and `jq`. It prints each Vegeta report, saves raw results and JSON metrics under the git-ignored `tmp/` directory, and exits with an error if any request fails or returns a status other than 200. A rate of `0` runs at full speed with 20 concurrent workers; set `VEGETA_WORKERS=50` to change that limit. Pass `http://127.0.0.1:9001/` as the URL to measure a mock backend directly with the same rates and duration.

## TODO

1. Implement load balancing strats
2. Create mock test servers with httptest covering various scenarios
3. Handle failover for test servers
4. Add health checks, error handling
5. Create monitoring tool (CLI or TUI based) showing state of load balancer
