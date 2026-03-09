package main

import (
	"fmt"
	"log"
	"net/http"

	"sheepy-wallet/internal/config"
	"sheepy-wallet/internal/handler"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Config.Host, cfg.Config.Port)
	log.Printf("starting sheepy-wallet on %s", addr)

	h := handler.New(cfg)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
