INGESTION_TRACE=./ingestion-trace.json
EXPORTED_TRACE=./exported-trace.json
NUMBER_OF_REQUESTS=2000

run:
	@rm $(EXPORTED_TRACE) || true
	@touch $(EXPORTED_TRACE)
	go run ./cmd/collector/* --config ./test-config.yml --log-level DEBUG

build:
	go build -o collector ./cmd/collector/* 

test:
	@curl -X POST http://localhost:9411/api/v2/spans \
  	-H "Content-Type: application/json" \
  	--data-binary "@$(INGESTION_TRACE)"

bench:
	ab -p $(INGESTION_TRACE) -T application/json -c 20 -n $(NUMBER_OF_REQUESTS) http://localhost:9411/api/v2/spans
	wc -l $(EXPORTED_TRACE)