package main

import (
	"flag"
	"fmt"

	MQTTClient "github.com/pablitovicente/mqtt-load-generator/pkg/MQTTClient"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Argument parsing
	messageCount := flag.Int("c", 1000, "Number of messages to send")
	messageSize := flag.Int("s", 100, "Size in bytes of the message payload")
	interval := flag.Int("i", 1, "Milliseconds to wait between messages")
	targetTopic := flag.String("t", "/load", "Target MQTT topic to publish messages to")
	username := flag.String("u", "", "MQTT username")
	password := flag.String("P", "", "MQTT password")
	host := flag.String("h", "localhost", "MQTT host")
	port := flag.Int("p", 1883, "MQTT port")
	numberOfClients := flag.Int("n", 1, "Number of concurrent MQTT clients")

	flag.Parse()

	// General Client Config
	mqttClientConfig := MQTTClient.Config{
		MessageCount: messageCount,
		MessageSize:  messageSize,
		Interval:     interval,
		TargetTopic:  targetTopic,
		Username:     username,
		Password:     password,
		Host:         host,
		Port:         port,
	}
	
	updates := make(chan int)

	pool := MQTTClient.Pool{
		SetupDone: make(chan struct{}),
		MqttClients: make([]MQTTClient.Client, 0),
	}

	pool.New(numberOfClients, mqttClientConfig, updates)
	<- pool.SetupDone
	fmt.Println("All clients connected....")

	pool.Start()

	bar := progressbar.Default(int64(*messageCount) * int64(*numberOfClients))

	go func (updates chan int) {
		for update := range updates {
			bar.Add(update)
		}
	}(updates)

	select {}
}
