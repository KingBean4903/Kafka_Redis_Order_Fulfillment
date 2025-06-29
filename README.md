## Kafka + Redis Order Deduplication PipelineKafka + Redis Order Deduplication Pipeline

This project demonstrates how to achieve exactly-once processing of events using Kafka and Redis for deduplication — without relying on Kafka Transactions.

We simulate a real-time order processing pipeline that can handle:

- Flash sale spikes
- Duplicate events
- Producer retries
- Malformed messages (sent to DLQ)

## Tech StackTech Stack		

- Go - high-performance services
- Kafka - durable, replayable event store
- Redis - in-memory deduplication using SETNX
- Docker Compose - local orchestration

## Getting Started

**1. Clone the repo**
`` git clone https://github.com/KingBean4903/Kafka_Redis_Order_Fulfillment.git 
	 cd Kafka_Redis_Order_Fulfillment
``
**2. Build & run services**
`` docker compose up --build ``

**3. Watch the magic happen**
- order-ingester sends a stream of fake and duplicate orders
- order-processor deduplicates and forwards valid ones.
- Malformed or failed messages go to the Dead Letter Queue (orders.dlq)

### How Deduplication Works

Each order is keyed by order_id. The processor:

1. Parses incoming Kafka message
2. Uses Redis SETNX to store dedup:order:<id> with a short TTL
3. If the key exists, it's a duplicate → skip
4. If new, validate and publish to orders.validated
5. Commit Kafka offset only if successful

Failures are sent to orders.dlq with a reason and timestamp.

### What’s Next

1. Add Prometheus + Grafana metrics
2. Retry logic with exponential backoff
3. Use Kafka Transactions to compare strategies

## Author
Built with joy by David Kanyango
