package rmqrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	Id        string
	queueName string
	channel   *amqp.Channel
	queue     *amqp.Queue
	messages  *<-chan amqp.Delivery
	router    map[string]func(ctx context.Context, delivery *Message)
	client    *Server
	stopChan  chan struct{}
}

func newConsumer(queue string, server *Server) *Consumer {
	return &Consumer{
		Id:        uuid.New().String(),
		queueName: queue,
		router:    make(map[string]func(ctx context.Context, delivery *Message)),
		client:    server,
	}
}

func (cons *Consumer) handleMessages() {
	for {
		select {
		case d, ok := <-*cons.messages:
			if !ok {
				fmt.Println("Channel messages closed, stopping handle")
				return
			}

			if len(d.Body) == 0 {
				fmt.Println("Received empty message, skipping")
				d.Ack(false)
				continue
			}

			msg := cons.makeMessage(&d)
			cons.handle(&msg)
		case <-cons.stopChan:
			fmt.Println("Received stop signal, stopping handle")
			return
		}
	}
}

func (cons *Consumer) writeMessage(msg *Message, payload []byte) {
	err := cons.channel.Publish(
		"",
		msg.delivery.ReplyTo,
		false, // обязательно
		false, // немедленно
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.delivery.CorrelationId,
			Body:          payload,
		})
	if err != nil {
		fmt.Println("Error publish message:", err)
		return
	}
}

func (cons *Consumer) handle(msg *Message) {

	ctx := context.Background()
	var req MessageReqEvent[any]
	err := json.Unmarshal(msg.delivery.Body, &req)
	if err != nil {
		fmt.Println("Error unmarshal request:", err)
		msg.SendError(http.StatusUnprocessableEntity, "Error unmarshal request", "UnprocessableEntity")
		return
	}

	fmt.Println("Received request:", req.Data.Subject)
	handler := cons.router[req.Data.Subject]

	if handler == nil {
		fmt.Println("Not found route pattern:", req.Data.Subject)
		msg.SendError(http.StatusNotFound, "Subject method not found", "NotFound")
		return
	}

	msg.SetPayload(req.Data.Payload)

	handler(ctx, msg)
}

func (c *Consumer) makeMessage(msg *amqp.Delivery) Message {
	return Message{
		consumer:  c,
		delivery:  msg,
		NeedReply: msg.ReplyTo != "",
	}
}

func (cons *Consumer) RegisterRoute(pattern string, handler func(ctx context.Context, delivery *Message)) {
	cons.router[pattern] = handler
	fmt.Println("Register pattern method:", pattern)
}

func (cons *Consumer) Start() {

	if cons.client.Connection.IsClosed() {
		fmt.Println("Connection is closed, cannot open a channel")
	}
	ch, err := cons.client.Connection.Channel()
	if err != nil {
		fmt.Println("Error get channel:", err)
		panic(err)
	}

	q, err := ch.QueueDeclare(
		cons.queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Println("Error queue declare:", err)
		panic(err)
	}

	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		fmt.Println("Failed to set QoS:", err)
		panic(err)
	}

	messages, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Println("Error listen channel:", err)
		panic(err)
	}

	cons.channel = ch
	cons.queue = &q
	cons.messages = &messages

	cons.stopChan = make(chan struct{})
	go cons.handleMessages()
}

func (cons *Consumer) Close() {
	cons.Stop()
	delete(cons.client.Consumers, cons.Id)
	fmt.Println("Consumer closed and deleted")
}

func (cons *Consumer) Stop() {
	close(cons.stopChan)

	if cons.channel != nil && !cons.channel.IsClosed() {
		err := cons.channel.Close()
		if err != nil {
			fmt.Println("Error closing channel:", err)
		}
		fmt.Println("Channel closed")
	} else {
		fmt.Println("Channel already closed")
	}

	fmt.Println("Consumer stopped")
}
