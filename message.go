package rmqrpc

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	consumer *Consumer
	delivery *amqp.Delivery
	payload  any
	ack      bool
}

func (msg *Message) SendResponse(payload any) {
	message := MessageResEvent[any]{
		Response: MessageResponse[any]{
			Response: &payload,
			Error:    nil,
		},
		IsDisposed: true,
	}

	pb, err := json.Marshal(message)
	if err != nil {
		msg.consumer.log.Error("Error marshal payload", err)
		pb = []byte(`{"status": "500", "error": "internal server error"}`)
	}

	msg.consumer.writeMessage(msg, pb)

	if !msg.ack {
		msg.delivery.Ack(false)
		msg.ack = true
	}
}

func (msg *Message) SendError(statusCode int, text string, name string) {
	message := MessageResEvent[any]{
		Response: MessageResponse[any]{
			Response: nil,
			Error: &MessageError{
				Status:  statusCode,
				Message: text,
				Name:    name,
			},
		},
		IsDisposed: true,
	}

	pb, err := json.Marshal(message)
	if err != nil {
		msg.consumer.log.Error("Error marshal payload", err)
		pb = []byte(`{"status": "500", "error": "internal server error"}`)
	}

	msg.consumer.writeMessage(msg, pb)

	if !msg.ack {
		msg.delivery.Ack(false)
		msg.ack = true
	}
}

func (msg *Message) Ack() bool {
	if msg.ack {
		return false
	}

	msg.delivery.Ack(false)
	msg.ack = true
	return true
}

func (msg *Message) GetValidatedPayload(target any) error {
	// Преобразуем msg.payload в JSON
	payloadBytes, err := json.Marshal(msg.payload)
	if err != nil {
		return err
	}

	// Десериализуем JSON в target
	err = json.Unmarshal(payloadBytes, target)
	if err != nil {
		return err
	}

	// Валидируем структуру target
	validator := validator.New()
	err = validator.Struct(target)
	if err != nil {
		return err
	}

	return nil
}

func (msg *Message) GetPayload() any {
	return msg.payload
}

func (msg *Message) SetPayload(payload any) {
	msg.payload = payload
}
