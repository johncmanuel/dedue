# dedue

dedue is a load balancer written in Go. Created for learning purposes!

## Run

Start local mock backends and the load balancer:
```bash
./run_cluster.sh

# visit the load balancer
curl -i http://127.0.0.1:8080/
```

The Go mock backends return their ID in the body and `X-Backend` header. Each also serves `GET /health` with a 200 status for the balancer's startup check. Press Ctrl-C in the cluster terminal to stop all servers. Use `./run_cluster.sh 3 9001 127.0.0.1:8080` to choose the backend count, first backend port, and listen address.

## TODO

1. Implement load balancing strats
2. Create mock test servers with httptest covering various scenarios
3. Handle failover for test servers
4. Add health checks, error handling
5. Use vegeta or something else to create mock traffic 
6. Create monitoring tool (CLI or TUI based) showing state of load balancer
