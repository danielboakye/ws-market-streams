package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/slog"
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

	logger *slog.Logger
}

type SubscriptionMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

const (
	subscriptionMethod   = "SUBSCRIBE"
	maxReconnectAttempts = 5

	BTCUSDT = "btcusdt"
)

func (ws *WebSocket) connect() error {
	// url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@depth", symbol)
	conn, _, err := websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/ws/depth", nil)
	if err != nil {
		return err
	}
	ws.conn = conn
	return nil
}

func NewWebSocketConnection(logger *slog.Logger, reconnect bool) (*WebSocket, error) {
	ws := &WebSocket{reconnect: reconnect, logger: logger}
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

func (ws *WebSocket) ReceiveUpdates(ctx context.Context, ch chan OrderBookUpdate) {
	var reconnectAttempts int
	backoffDuration := 5 * time.Second

	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			ws.logger.Info(fmt.Sprintf("Error reading message: %v", err))

			if !ws.reconnect || reconnectAttempts >= maxReconnectAttempts {
				close(ch)
				return
			}

			select {
			case <-time.After(backoffDuration):
				err := ws.Reconnect()
				if err != nil {
					reconnectAttempts++
					backoffDuration *= 2
					ws.logger.Warn(fmt.Sprintf("Reconnection attempt %d failed: %v", reconnectAttempts, err))
					continue
				}

				err = ws.Subscribe(ws.symbol)
				if err != nil {
					reconnectAttempts++
					backoffDuration *= 2
					ws.logger.Warn(fmt.Sprintf("Subscription attempt %d for %s failed: %v", reconnectAttempts, ws.symbol, err))
					continue
				}

				reconnectAttempts = 0
				backoffDuration = 5 * time.Second

			case <-ctx.Done():
				ws.logger.Info("Received shutdown signal, exiting...")
				close(ch)
				return
			}
		}

		var update OrderBookUpdate
		err = json.Unmarshal(data, &update)
		if err != nil {
			ws.logger.Info(fmt.Sprintf("Error un marshalling data: %v", err))
			continue
		}

		select {
		case ch <- update:
			ws.logger.Info("new data received")
		default:
			ws.logger.Warn("No receiver for update, discarding")
		}
	}
}

func (ws *WebSocket) Reconnect() error {
	return ws.connect()
}

func (ws *WebSocket) Close() error {
	return ws.conn.Close()
}
