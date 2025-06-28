package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"string"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
)

var (

	topic = "orders.placed"
	redisDedupPrefix = "dedup:order:"
	dedupTTL = 5 * time.Minute
)

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

				orderID := extractOrderID(string(msg.Value))
				redisKey := redisDedupPrefix + orderID

				ok, err := rdb.SETNX(ctx, redisKey, "1", dedupTTL).Result()
				if err != nil {
						log.Printf("Redis error: %v", err)
						continue
				}

				log.Printf("Processing order %s", orderID)
				// TODO validate order
				
				_, err := consumer.CommitMessage(msg)
				if err != nil {
						log.Printf("Offset commit failed: %v", err)
				}		
		}

	}
}

func extractOrderID(value string) string {
	
	return strings.TrimSpace(value)
}

func getENV(key, fallback string) string {
		
	if v := os.Getenv(key); v != " " {
			return v
	}

	return fallback

}
