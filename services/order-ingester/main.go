package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Order struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
	SKU     string `json:"sku"`
	Qty     int `json:"qty"`
}

func main() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	rand.Seed(time.Now().UnixNano())

	// brokers := []string{"kafka1:9093"}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": getENV("KAFKA_BOOTSTRAP", "localhost:9092"),
	})

	if err != nil {
			log.Fatalf("Failed to create Producer, %v", err)
	}

	defer producer.Close()

	topic := "order.placed"
	quit  := make(chan bool)

	go func() {
		<- sigs
		fmt.Println("\n Shutting down ingester...")
		quit <- true
	}()

	fmt.Println("Order ingester started. Sending fake orders..")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <- quit:
				return
		case <- ticker.C:
				order := generateFakeOrder(count)
				orderBytes, _ := json.Marshal(order)

				err = producer.Produce(&kafka.Message{
							TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
							Value: orderBytes,
				}, nil)

				if err != nil {
						log.Printf("Failed to produce: %v", err)
				} else {
						log.Printf("Sent order: %s", order.OrderID)
				}

				// Occassionally send a  duplicate
				if count%10 == 0 {
						log.Printf("Sending duplicate for: %s", order.OrderID)
						producer.Produce(&kafka.Message{
							TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
							Value: orderBytes,
						}, nil)
				}
				count++
		}
	}
}

func generateFakeOrder(n int) Order {
	
	return Order{
			OrderID : fmt.Sprintf("order-%d", n),
			UserID   : fmt.Sprintf("user-%d", rand.Intn(100)),
			SKU : fmt.Sprintf("sku-%d", rand.Intn(20)),
			Qty : rand.Intn(5) + 1,
	}
}


func getENV(key, fallback string) string {
	
	if v := os.Getenv(key); v != " " { 
				return v
	}

 	 return fallback
}
