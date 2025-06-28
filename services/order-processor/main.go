package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
)

var (
	topic = "orders.placed"
	redisDedupPrefix = "dedup:order:"
	dedupTTL = 5 * time.Minute
	outputTopic = "orders.validate"
)

type Order struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
	SKU     string `json:"sku"`
	Qty     string `json:"qty"`
}


func main() {
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		 	<- sigs
			fmt.Println("\n Shutting down ... ")
			cancel()
	}()
	
	rdb := redis.NewClient(&redis.NewClient(
			Addr: getENV("REDIS_ADDR", "localhost:6379")
			Password: " ".
			DB: 0,
	))

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
					"bootstrap.servers": getENV("KAFKA_BOOTSTRAP", "localhost:9092"),
					"group.id" : "order.processor",
					"enable.auto.commit": false,
					"auto.offset.reset" : "earliest".
	})

	if err != nil {
		log.Fatalf("Failed to create consumer:", %v)
	}

	defer consumer.Close()
	
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers" : getENV("KAFKA_BOOTSTRAP", "localhost:9092"),
	})

	if err != nil {
			log.Fatalf("Failed to create producer: %v", err)
	}
	
	defer producer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)

	if err != nil {
		log.Fatalf("Failed to subscribe:  %v", err)
	}

	fmt.Println("Order processing started. Listening for messages ... ")


	for {
		
		select {
		case <- ctx.Done():
				return
		default:
				msg, err := consumer.ReadMessage(500 * time.Millisecond)
				if err != nil {
						if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrorTimeOut {
								continue
						}
						log.Printf("consumer err: %v", err)
						continue
				}

	//			orderID := extractOrderID(string(msg.Value))
		
				order, err := parseOrder(msg.Value)
				if err != nil {
						log.Printf("Invalid order format: %v", err)
						sendToDLQ(producer, msg.Value, fmt.Sprintf("parse error: %v", err))
						continue
				}

				redisKey := redisDedupPrefix + orderID
				ok, err := rdb.SETNX(ctx, redisKey, "1", dedupTTL).Result()
				if err != nil {
						log.Printf("Redis error: %v", err)
						sendToDLQ(producer, msg.Value, fmt.Sprintf("parse error: %v", err))
						continue
				}

				if !ok {
						log.Printf("Duplicate order %s skipped", order.OrderID)
						consumer.CommitMessage(msg)
						continue
				}

				log.Printf("Validating and forwarding order: %s", order.OrderID)
				orderBytes, _ := json.Marshal(order)
				err = producer.Produce(&kafka.Message{
							TopicPartition: kafka.TopicPartition{Topic: &outputTopic, Partition: kafka.PartitionAny},
							Value: orderBytes
				}, nil)
				if err != nil {
						log.Printf("Failed to publish order: %v", err)
						sendToDLQ(producer, msg.Value, fmt.Sprintf("parse error: %v", err))
						continue
				}
			
				_, err := consumer.CommitMessage(msg)
				if err != nil {
						log.Printf("Offset commit failed: %v", err)
				}		
		}

	}
}

func sendToDLQ(p *kafka.Producer, original []byte, reason string) { 
	
	dlqMessage := map[string]interface{} {
			"timestamp" : time.Now().UTC().Format(time.RFC3339),
			"original" : string(original),
			"reason": reason,
	}
	dlqBytes, _ := json.Marshal(dlqMessage)

	_ = p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &dlqTopic, Partition: kafka.PartitionAny},
				Value: dlqBytes,
	}, nil)

}


// Function to parse order
func parseOrder(data []byte) (*Order, error) {
		
	var o Order
	err := json.Unmarshal(data, &o)
	if err != nil || strings.TrimSpace(o.OrderID) == " " {
			return nil, fmt.Errorf("Invalid order payload")
	}
	return &o, nil

}

func getENV(key, fallback string) string {
		
	if v := os.Getenv(key); v != " " {
			return v
	}

	return fallback

}
