# dedue

[dedue](https://fireemblem.fandom.com/wiki/Dedue) is a load balancer written in Go. Created for learning purposes! Currently supports static load balancing at the moment.

## Run

Start local mock backends and the load balancer:
```bash
./scripts/run_cluster.sh

# visit the load balancer
curl -i http://127.0.0.1:8080/
```

The Go mock backends return their ID in the body and `X-Backend` header. Each also serves `GET /health` with a 200 status. The balancer checks backend health at startup and every minute afterward (will later be configurable via CLI flags).

The default strategy is `round-robin`. To use another strategy like `random`, run `./scripts/run_cluster.sh 3 9001 127.0.0.1:8080 random`, or pass `-strategy=random` to `go run .` before the backend URLs.

Pass backend URLs directly to `go run .` to configure a static server list. Add `,<weight integer>` to set a positive weight for a backend; plain URLs default to weight 1:

```bash
go run . http://127.0.0.1:9001,3 http://127.0.0.1:9002,1
```

## Supported LB strategies

1. `round-robin`
2. `random`
3. `weighted-round-robin`

## Real time load testing

Install [Vegeta](https://github.com/tsenart/vegeta), then start the cluster with `./scripts/run_cluster.sh`. As an example, spin up a second terminal and send 100 requests per second for 30 seconds:

```bash
printf 'GET http://127.0.0.1:8080/\n' | vegeta attack -rate=100 -duration=30s | vegeta report
```

Repeat with higher `-rate` values (for example, 500 and 1000). Compare the achieved request rate, errors, and latency percentiles as load increases. To estimate the balancer's overhead, run the same command against a mock backend directly by replacing port `8080` with `9001` (the default first backend port). Keep the rate and duration the same for both runs.

For a repeatable run, keep `./scripts/run_cluster.sh` running and, in another terminal, run:

```bash
./scripts/load_test.sh                                    # 100, 500, and 1000 requests/s for 30s each
./scripts/load_test.sh http://127.0.0.1:8080/ 5s 20 100   # shorter custom run
./scripts/load_test.sh http://127.0.0.1:8080/ 5s 0        # unlimited rate with 20 workers
```

The script requires Vegeta and `jq`. It prints each Vegeta report, saves raw results and JSON metrics under the git-ignored `tmp/` directory, and exits with an error if any request fails or returns a status other than 200. A rate of `0` runs at full speed with 20 concurrent workers. Use `VEGETA_WORKERS=<number of workers>` to change that limit. Pass `http://127.0.0.1:9001/` as the URL to measure a mock backend directly with the same rates and duration.

## TODO

1. Implement load balancing strats
2. Create mock test servers with httptest covering various scenarios
3. Handle failover for test servers
4. Add health checks, error handling
5. Create monitoring tool (CLI or TUI based) showing state of load balancer

## Learning sources used

1. https://www.cloudflare.com/learning/performance/types-of-load-balancing-algorithms/
2. https://www.youtube.com/watch?v=PERKHUJYotM&list=PLjTveVh7FakLdTmm42TMxbN8PvVn5g4KJ&index=58
