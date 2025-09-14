run-lb:
	go run ./cmd/lb
run-worker:
	go run ./cmd/worker
bench:
	go run ./scripts/loadgen.go