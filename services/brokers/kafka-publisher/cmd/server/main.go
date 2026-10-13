package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	publisher "github.com/manny-ovena/microendpoint-stack/services/brokers/kafka-publisher"
)

const (
	envListenAddress = "KAFKA_PUBLISHER_ADDR"
	defaultListen    = ":9000"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (returnErr error) {
	service, err := publisher.NewFromEnv()
	if err != nil {
		return fmt.Errorf("create Kafka publisher: %w", err)
	}
	defer func() {
		if err := service.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close Kafka publisher: %w", err))
		}
	}()

	address := strings.TrimSpace(os.Getenv(envListenAddress))
	if address == "" {
		address = defaultListen
	}
	server, err := publisher.NewTCPServer(address, service)
	if err != nil {
		return fmt.Errorf("create publisher RPC server: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("Kafka publisher RPC listening on %s", server.Addr())
	if err := server.Serve(ctx); err != nil {
		return fmt.Errorf("serve publisher RPC: %w", err)
	}
	return nil
}
