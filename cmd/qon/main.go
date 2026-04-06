package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/qonhq/qon/internal/bridge"
	"github.com/qonhq/qon/internal/config"
	"github.com/qonhq/qon/internal/core"
	"github.com/qonhq/qon/internal/server"
)

func main() {
	var mode string
	var addr string
	var accessKey string
	flag.StringVar(&mode, "mode", "bridge", "execution mode: bridge or server")
	flag.StringVar(&addr, "addr", ":8080", "server mode listen address")
	flag.StringVar(&accessKey, "access-key", "", "optional access key for request authorization")
	flag.Parse()

	cfg := config.Default()
	cfg.AccessKey = accessKey
	client := core.NewClient(cfg)
	defer client.Close()

	switch mode {
	case "bridge":
		if err := bridge.RunJSONStdio(client, os.Stdin, os.Stdout); err != nil {
			log.Fatalf("bridge failed: %v", err)
		}
	case "server":
		srv := server.New(client)
		log.Printf("qon server listening on %s", addr)
		if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	default:
		log.Fatalf("unknown mode: %s", mode)
	}
}
