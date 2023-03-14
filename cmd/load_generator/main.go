package main

import (
	"flag"
	"fmt"
	"sync"

	MQTTClient "github.com/pablitovicente/mqtt-load-generator/pkg/MQTTClient"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Argument parsing
	messageCount := flag.Int("c", 1000, "Number of messages to send")
	messageSize := flag.Int("s", 100, "Size in bytes of the message payload")
	interval := flag.Int("i", 1, "Milliseconds to wait between messages")
	schedule := flag.String("z", "normal", "Distribution of time between messages: 'flat': always wait Interval between messages, 'normal': wait a random amount between messages with mean equal to the interval and stdev to half interval, 'random': wait a random amount between messages with mean equal to the interval.")
	targetTopic := flag.String("t", "/load", "Target MQTT topic to publish messages to")
	username := flag.String("u", "", "MQTT username")
	password := flag.String("P", "", "MQTT password")
	host := flag.String("h", "localhost", "MQTT host")
	port := flag.Int("p", 1883, "MQTT port")
	numberOfClients := flag.Int("n", 1, "Number of concurrent MQTT clients")
	idAsSubTopic := flag.Bool("suffix", false, "If set to true integers will be used as sub-topic to the topic specified by 't'. The range goes from 1 to N where N is the max number of configured concurrent clients.")
	qos := flag.Int("q", 1, "MQTT QoS used by all clients")
	cert := flag.String("cert", "", "Path to TLS certificate file")
	ca := flag.String("ca", "", "Path to TLS CA file")
	key := flag.String("key", "", "Path to TLS key file")
	insecure := flag.Bool("insecure", false, "Set to true to allow self signed certificates")
	mqtts := flag.Bool("mqtts", false, "Set to true to use MQTTS")

	flag.Parse()

	if *qos < 0 || *qos > 2 {
		panic("QoS should be any of [0, 1, 2]")
	}

	// General Client Config
	mqttClientConfig := MQTTClient.Config{
		MessageCount: messageCount,
		MessageSize:  messageSize,
		Interval:     interval,
		Schedule:     schedule,
		TargetTopic:  targetTopic,
		Username:     username,
		Password:     password,
		Host:         host,
		Port:         port,
		IdAsSubTopic: idAsSubTopic,
		QoS:          qos,
		Insecure:     insecure,
		MQTTS:        mqtts,
	}
	// If ca, cert, and key were set configure TLS
	if TLSOptionsSet() {
		mqttClientConfig.TLSConfigured = true
		mqttClientConfig.CA = ca
		mqttClientConfig.Cert = cert
		mqttClientConfig.Key = key
	}

	updates := make(chan int)

	pool := MQTTClient.Pool{
		SetupDone:   make(chan struct{}),
		MqttClients: make([]*MQTTClient.Client, 0),
	}
	fmt.Printf("Setting up %d MQTT clients\n", *numberOfClients)
	pool.New(numberOfClients, mqttClientConfig, updates)
	// Wait until all the setup is done
	<-pool.SetupDone
	fmt.Println("All clients connected, starting publishing messages")
	var wg sync.WaitGroup
	pool.Start(&wg)

	bar := progressbar.Default(int64(*messageCount) * int64(*numberOfClients))

	go func(updates chan int) {
		for update := range updates {
			bar.Add(update)
		}
	}(updates)

	wg.Wait()
	// Hacky way of avoiding the progress bar going away.
	// Todo: check why this happens
	bar.Add(0)
}

func TLSOptionsSet() bool {
	foundCert := false
	foundCA := false
	foundKey := false

	flag.Visit(func(f *flag.Flag) {
		if f.Name == "cert" {
			foundCert = true
		}

		if f.Name == "ca" {
			foundCA = true
		}

		if f.Name == "key" {
			foundKey = true
		}
	})

	return foundCA && foundCert && foundKey
}
