package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ws-binance/internal/ws/binance"
	"golang.org/x/exp/slog"
)

var (
	reconnectWebsocketFlag bool
)

func init() {
	flag.BoolVar(&reconnectWebsocketFlag, "reconnect", true, "Enable or disable automatic reconnection to the WebSocket.")
}

func main() {
	flag.Parse()

	binanceWS, err := binance.NewWebSocketConnection(slog.New(slog.NewTextHandler(os.Stdout, nil)), reconnectWebsocketFlag)
	if err != nil {
		log.Fatal("Error creating WebSocket:", err)
	}
	defer binanceWS.Close()

	err = binanceWS.Subscribe(binance.BTCUSDT)
	if err != nil {
		log.Fatal("Error subscribing:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates := make(chan binance.OrderBookUpdate, 100)
	go binanceWS.ReceiveUpdates(ctx, updates)

	go func() {
		for update := range updates {
			fmt.Printf("Received update: %v\n", update)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-signalChan
	cancel()

}
