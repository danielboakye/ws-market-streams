// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/ws-binance/internal/ws/binance"
)

var (
	reconnectWebsocketFlag bool
	port                   string
)

func init() {
	flag.BoolVar(&reconnectWebsocketFlag, "reconnect", true, "Enable or disable automatic reconnection to the WebSocket.")
	flag.StringVar(&port, "port", "8080", "Web server port")
}

func main() {
	flag.Parse()

	binanceWS, err := binance.NewWSConnection(reconnectWebsocketFlag)
	if err != nil {
		log.Fatal("Error creating WebSocket:", err)
	}
	defer binanceWS.Close()

	err = binanceWS.Subscribe("btcusdt")
	if err != nil {
		log.Fatal("Error subscribing:", err)
	}

	updates := make(chan binance.OrderBookUpdate)
	go binanceWS.ReceiveUpdates(updates)

	for update := range updates {
		fmt.Printf("Received update: %v\n", update)
	}

	// Handle graceful shutdown on Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	binanceWS.Close()
	fmt.Println("Shutting down...")

	// Keep the main Goroutine running to receive updates.
	// select {}
}
