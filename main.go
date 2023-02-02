package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	x := initLogging(9)
	topic := "topic"
	defer x()
	tracer := otel.Tracer("nats-echo-client")
	ctx := context.Background()
	natsUrl := nats.DefaultURL
	Debug("Connection to nats", "url", natsUrl)
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		Error(err, "failed to connect to nats", "url", natsUrl)
	}
	//Info("Connected to nats", "ConnectedUrl", nc.ConnectedUrl(), "Servers", nc.Servers())
	sub, err := nc.Subscribe(topic, func(m *nats.Msg) {
		id := uuid.New().String()
		_, span := tracer.Start(context.WithValue(ctx, "id", id), "nats-message")
		defer span.End()
		span.AddEvent("Received message")
		Info("Received a message", "headers", m.Header, "body", string(m.Data))
		span.AddEvent("Message done message")

	})
	if err != nil {
		Error(err, "failed to subscribe to topic", "topic", topic)
		return
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	Info("Blocking, press ctrl+c to continue...")
	<-done
	Debug("Process interrupted, unsubscribing from topic", "topic", topic)
	// Unsubscribe
	err = sub.Unsubscribe()
	if err != nil {
		Error(err, "Failed to unsubscribe")
	}

	// Drain
	Debug("Draining subscription", "topic", topic)
	err = sub.Drain()
	if err != nil {
		Error(err, "Failed to drain")
	}
	// Close connection
	nc.Close()
	log.Println("Closed connection")
}
