package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	schemaregistry "github.com/lensesio/schema-registry"
	"github.com/timvw/kafkaavro"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	kafkaConfig := &kafka.ConfigMap{
		"metadata.broker.list": "localhost:9092",
		"group.id":             "go-test2",
		"auto.offset.reset":    "earliest",
		"enable.auto.commit":   false,
	}

	schemaRegistryURL := "http://localhost:8081"

	client, err := schemaregistry.NewClient(schemaRegistryURL)
	if err != nil {
		return
	}

	topic := "test"

	subjectNameStrategy := kafkaavro.TopicNameStrategy{}
	subjectName := subjectNameStrategy.GetSubjectName(topic, false)

	valueDecoder, err := kafkaavro.NewDecoder(*client, subjectName)
	if err != nil {
		panic(err)
	}

	kafkaConsumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		panic(err)
	}

	kafkaConsumer.SubscribeTopics([]string{topic}, nil)

	run := true

	for run == true {

		select {

		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false

		default:

			ev := kafkaConsumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {

			case *kafka.Message:

				if(len(e.Value)==0){
					fmt.Printf("Message was null. On a log-compacted topic this means a delete")
					break
				}

				native, err := valueDecoder.Decode(e.Value)

				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Message on %s: %s\n", e.TopicPartition, native)
				}

			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				// But in this example we choose to terminate
				// the application if all brokers are down.
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					run = false
				}

			default:
				fmt.Printf("Ignored %v\n", e)
			}

		}
	}

	fmt.Printf("Closing consumer\n")
	kafkaConsumer.Close()

}
