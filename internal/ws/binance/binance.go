package binance

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type OrderBookUpdate struct {
	EventType string     `json:"e"`
	EventTime int64      `json:"E"`
	Symbol    string     `json:"s"`
	UpdateID  int64      `json:"u"`
	Bids      [][]string `json:"b"`
	Asks      [][]string `json:"a"`
}

type WebSocket struct {
	conn      *websocket.Conn
	symbol    string
	reconnect bool
}

type SubscriptionMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

const subscriptionMethod = "SUBSCRIBE"

func (ws *WebSocket) connect() error {
	// url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@depth", symbol)
	conn, _, err := websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/ws/depth", nil)
	if err != nil {
		return err
	}
	ws.conn = conn
	return nil
}

func NewWebSocketConnection(reconnect bool) (*WebSocket, error) {
	ws := &WebSocket{reconnect: reconnect}
	if err := ws.connect(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (ws *WebSocket) Subscribe(symbol string) error {
	ws.symbol = symbol
	subscription := &SubscriptionMessage{
		Method: subscriptionMethod,
		Params: []string{fmt.Sprintf("%s@depth", symbol)},
		ID:     1,
	}

	message, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	return ws.conn.WriteMessage(websocket.TextMessage, message)
}

func (ws *WebSocket) ReceiveUpdates(ch chan OrderBookUpdate) {
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)

			if !ws.reconnect {
				close(ch)
				return
			}
			time.Sleep(5 * time.Second)
			if err := ws.Reconnect(); err == nil {
				_ = ws.Subscribe(ws.symbol)
			}
		}

		// fmt.Println("Received raw data:", string(data)) // Print raw data

		var update OrderBookUpdate
		err = json.Unmarshal(data, &update)
		if err != nil {
			log.Println("Error un marshalling data:", err)
			continue
		}

		// fmt.Printf("Update received: %v\n", update)

		// Send the update to the provided channel
		ch <- update
	}
}

func (ws *WebSocket) Reconnect() error {
	return ws.connect()
}

func (ws *WebSocket) Close() error {
	return ws.conn.Close()
}
