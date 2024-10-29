package rmqrpc

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Server struct {
	Url        string
	Connection *amqp.Connection
	Consumers  map[string]*Consumer
}

func NewAMQPServer(opts AMQPServerOpts) *Server {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", opts.RmqUser, opts.RmqPass, opts.RmqHost, opts.RmqPort)

	conn, err := amqp.Dial(url)
	if err != nil {
		fmt.Println("Error connect AMQP:", err)
		panic(err)
	}

	fmt.Println("AMQP server connected")

	return &Server{
		Url:        url,
		Connection: conn,
		Consumers:  make(map[string]*Consumer),
	}
}

func (c *Server) Close() {
	for _, cons := range c.Consumers {
		cons.Close()
	}

	fmt.Println("Consumers closed")

	if c.Connection != nil && !c.Connection.IsClosed() {
		err := c.Connection.Close()
		if err != nil {
			fmt.Println("Error close AMQP:", err)
		}
		fmt.Println("AMQP connection closed")
	} else {
		fmt.Println("AMQP connection already closed")
	}

	fmt.Println("AMQP closed")
}

func (c *Server) NewConsumer(queue string) *Consumer {
	cons := newConsumer(queue, c)
	c.Consumers[cons.Id] = cons

	fmt.Println("New Consumer Register:", queue)
	return cons
}

func (c *Server) QueueDelete(queue string) error {
	ch, err := c.Connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDelete(queue, false, false, false)
	if err != nil {
		return err
	}

	return nil
}

func (c *Server) Start() {
	for _, cons := range c.Consumers {
		go cons.Start()
	}
	fmt.Println("Started")
}

func (c *Server) Run(reconnect bool) {
	if reconnect {
		go c.WatchConnection()
	}

	c.Start()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-stopChan
	}()

	wg.Wait()
	fmt.Println("Server stopped")
}

func (c *Server) PurgeQueue(queue string) error {
	ch, err := c.Connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueuePurge(queue, true)
	if err != nil {
		return err
	}

	return nil
}

func (c *Server) WatchConnection() {
	notifyClose := c.Connection.NotifyClose(make(chan *amqp.Error))

	for {
		err := <-notifyClose
		if err == nil {
			break
		}

		fmt.Printf("AMQP error connection: %s\n", err)

		for _, cons := range c.Consumers {
			cons.Stop()
		}

		for {
			newConn, err := c.reconnect()
			if err == nil {
				c.Connection = newConn
				notifyClose = c.Connection.NotifyClose(make(chan *amqp.Error))
				fmt.Println("AMQP connection restored")

				for _, cons := range c.Consumers {
					cons.Start()
				}

				break
			}

			fmt.Printf("Error reconnect: %s\n", err)

			for i := 5; i > 0; i-- {
				fmt.Printf("Error reconnect: Try again in %d seconds...\n", i)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (c *Server) reconnect() (*amqp.Connection, error) {
	return amqp.Dial(c.Url)
}
