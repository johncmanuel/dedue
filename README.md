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

## Real time load testing

Install [Vegeta](https://github.com/tsenart/vegeta), then start the cluster with `./run_cluster.sh`. In a second terminal, send 100 requests per second for 30 seconds:

```bash
printf 'GET http://127.0.0.1:8080/\n' | vegeta attack -rate=100 -duration=30s | vegeta report
```

Repeat with higher `-rate` values (for example, 500 and 1000). Compare the achieved request rate, errors, and latency percentiles as load increases. To estimate the balancer's overhead, run the same command against a mock backend directly by replacing port `8080` with `9001` (the default first backend port). Keep the rate and duration the same for both runs.

## TODO

1. Implement load balancing strats
2. Create mock test servers with httptest covering various scenarios
3. Handle failover for test servers
4. Add health checks, error handling
5. Create monitoring tool (CLI or TUI based) showing state of load balancer
