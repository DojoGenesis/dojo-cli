package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DojoGenesis/dojo-cli/internal/client"
	"github.com/DojoGenesis/dojo-cli/internal/config"
	"github.com/DojoGenesis/dojo-cli/internal/repl"
	"github.com/fatih/color"
)

const version = "0.1.0"

func main() {
	var (
		flagGateway     = flag.String("gateway", "", "Gateway URL (overrides config, e.g. http://localhost:7340)")
		flagToken       = flag.String("token", "", "Bearer token for gateway auth")
		flagVersion     = flag.Bool("version", false, "Print version and exit")
		flagNoColor     = flag.Bool("no-color", false, "Disable color output")
	)
	flag.Parse()

	if *flagNoColor {
		color.NoColor = true
	}

	if *flagVersion {
		fmt.Printf("dojo %s\n", version)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fatalf("config error: %s", err)
	}

	// Flag overrides
	if *flagGateway != "" {
		cfg.Gateway.URL = *flagGateway
	}
	if *flagToken != "" {
		cfg.Gateway.Token = *flagToken
	}

	// Ensure ~/.dojo exists
	if err := os.MkdirAll(config.DojoDir(), 0700); err != nil {
		fatalf("could not create ~/.dojo: %s", err)
	}

	// Build gateway client
	gw := client.New(cfg.Gateway.URL, cfg.Gateway.Token, cfg.Gateway.Timeout)

	// Cancellable context — catches Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run REPL (plugin scan happens inside repl.New)
	r := repl.New(cfg, gw)
	if err := r.Run(ctx); err != nil {
		fatalf("repl error: %s", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dojo: "+format+"\n", args...)
	os.Exit(1)
}
