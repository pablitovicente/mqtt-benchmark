package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	MQTTClient "github.com/pablitovicente/mqtt-load-generator/pkg/MQTTClient"
	"github.com/paulbellamy/ratecounter"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Argument parsing
	targetTopic := flag.String("t", "/load", "Target MQTT topic to publish messages to")
	username := flag.String("u", "", "MQTT username")
	password := flag.String("P", "", "MQTT password")
	host := flag.String("h", "localhost", "MQTT host")
	port := flag.Int("p", 1883, "MQTT port")
	qos := flag.Int("q", 1, "MQTT QoS used by all clients")
	disableBar := flag.Bool("disable-bar", false, "Disable interactive mode to display statistics as log messages instead of interactive output")
	resetTime := flag.Float64("reset-after", 30, "Reset counter after <n> seconds without a message")

	flag.Parse()

	if *qos < 0 || *qos > 2 {
		panic("QoS should be any of [0, 1, 2]")
	}

	if !*disableBar {
		fmt.Println("press ctrl+c to exit")
	}

	// General Client Config
	mqttClientConfig := MQTTClient.Config{
		TargetTopic: targetTopic,
		Username:    username,
		Password:    password,
		Host:        host,
		Port:        port,
		QoS:         qos,
	}

	rand.Seed(time.Now().UnixNano())
	updates := make(chan int)

	mqttClient := MQTTClient.Client{
		ID:      rand.Intn(100000),
		Config:  mqttClientConfig,
		Updates: updates,
	}

	mqttClient.Connect()

	mqttClient.Subscribe(*targetTopic)
	if !*disableBar {
		bar := progressbar.Default(-1)
		go func(updates chan int) {
			for update := range updates {
				bar.Add(update)
			}
		}(updates)

		// There's some issue with bar update when traffic is not constant
		// so this go routine updates the bar with 0 just to get the total numbers right
		ticker := time.NewTicker(1 * time.Second)
		go func() {
			for {
				// Block until the clock ticks
				<-ticker.C
				// Update bar with 0 to update total
				bar.Add(0)
			}
		}()
	} else {
		// Store total number of received messages since start or last reset
		msgCount := 0
		// Create a rate counter, that holds the number of messages per second
		rateCounter := ratecounter.NewRateCounter(1 * time.Second)
		tickTime := time.Now()
		go func(updates chan int) {
			for update := range updates {
				// Add the number of received msgs to the current total
				msgCount += update
				// Increase the rate counter by the number of received messages for the current tick
				rateCounter.Incr(int64(update))
				// Mark the last time we received a message
				tickTime = time.Now()
			}
		}(updates)

		uptimeTicker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-uptimeTicker.C:
				// Every second, as long as there are messages being received
				if msgCount > 0 {
					// output the total number of messages and the current rate
					fmt.Printf("Received %d messages so far while handling %d msg/sec\n", msgCount, rateCounter.Rate())
					if time.Since(tickTime).Seconds() > *resetTime {
						// last received message is too long ago, reset counter and stop log messages
						fmt.Printf("Did not receive a message for at least %d seconds. Resetting counter.\n", int(*resetTime))
						fmt.Println("Log will continue when new messages arrive.")
						msgCount = 0
					}
				}
			}
		}
	}
	select {}
}
