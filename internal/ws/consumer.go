package ws

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/benniu/tradeengine/internal/kafka"
	"github.com/benniu/tradeengine/internal/models"
)

// StartExecutionConsumer reads from the "executions" Kafka topic and pushes
// real-time updates to connected WebSocket clients via the Hub.
func StartExecutionConsumer(ctx context.Context, brokers []string, hub *Hub) {
	consumer := kafka.NewConsumer(brokers, "executions", "ws-broadcaster")
	defer consumer.Close()

	log.Println("ws: execution consumer started")

	consumer.Consume(ctx, func(msg kafkago.Message) error {
		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("ws: bad execution message: %v", err)
			return nil // skip poison pill
		}

		order := event.Order

		// Send order update to the order's owner
		hub.SendToUser(order.UserID, WSMessage{
			Type: "order_update",
			Data: order,
		})

		// Send trade notifications to both sides
		for _, t := range event.Trades {
			tradeMsg := WSMessage{Type: "trade", Data: t}

			// Notify buyer
			buyOrder, _ := findUserForOrder(event, t.BuyOrderID)
			if buyOrder != nil {
				hub.SendToUser(buyOrder.UserID, tradeMsg)
			}

			// Notify seller
			sellOrder, _ := findUserForOrder(event, t.SellOrderID)
			if sellOrder != nil && (buyOrder == nil || sellOrder.UserID != buyOrder.UserID) {
				hub.SendToUser(sellOrder.UserID, tradeMsg)
			}
		}

		return nil
	})
}

// findUserForOrder is a helper — since OrderEvent only has the submitting order,
// we broadcast trade notifications to both sides by checking the trade's order IDs.
// For the submitting user we already have the info; for the counterparty we rely
// on the trade record containing BuyUserID/SellUserID (not currently in Trade model).
// For now, we send trade updates only to the submitting user.
func findUserForOrder(event models.OrderEvent, orderID uuid.UUID) (*models.Order, error) {
	if event.Order.ID == orderID {
		return &event.Order, nil
	}
	return nil, nil
}
