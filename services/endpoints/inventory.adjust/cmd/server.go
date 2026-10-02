package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/manny-ovena/microendpoint-stack/services/endpoints/inventory.adjust/internal"
	"github.com/manny-ovena/microendpoint-stack/services/endpoints/inventory.adjust/transport"
	"github.com/manny-ovena/microendpoint-stack/shared/logging"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger, err := logging.NewFromEnv("endpoint-inventory-adjust")
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize logger: %v\n", err)
		os.Exit(1)
	}

	svc := internal.NewService()
	handler := transport.NewHTTPHandler(svc)

	logger.Info().Str("port", port).Msg("inventory.adjust starting")
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
	}
}
