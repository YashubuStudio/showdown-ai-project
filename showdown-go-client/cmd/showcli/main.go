package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ysu/showdown-go-client/internal/gui"
	"github.com/ysu/showdown-go-client/pkg/showdown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "ping":
		runPing(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "mockbattle":
		runMockBattle(os.Args[2:])
	case "serve":
		runServe(os.Args[2:], false)
	case "gui":
		runServe(os.Args[2:], true)
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`showcli <command>

Commands:
  ping        Check websocket connection and local rename flow
  status      Fetch server room status
  mockbattle  Run a simple automated random battle
  serve       Start local HTTP API + GUI without auto-opening browser
  gui         Start local HTTP API + GUI and open browser`)
}

func runPing(args []string) {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	username := fs.String("username", "cli-ping", "Username prefix")
	fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := showdown.Ping(ctx, *server, *username)
	exitIf(err)
	printJSON(info)
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	username := fs.String("username", "cli-status", "Username prefix")
	fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	status, err := showdown.FetchStatus(ctx, *server, *username)
	exitIf(err)
	printJSON(status)
}

func runMockBattle(args []string) {
	fs := flag.NewFlagSet("mockbattle", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	format := fs.String("format", "gen9randombattle", "Battle format")
	timeout := fs.Duration("timeout", 90*time.Second, "Battle timeout")
	fs.Parse(args)

	ctx := context.Background()
	result, err := showdown.RunMockBattle(ctx, showdown.MockBattleRequest{
		ServerURL: *server,
		Format:    *format,
		Timeout:   *timeout,
	})
	exitIf(err)
	printJSON(result)
}

func runServe(args []string, openBrowser bool) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8099", "Listen address")
	fs.Parse(args)

	server := &http.Server{
		Addr:              *addr,
		Handler:           gui.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", *addr)
	exitIf(err)
	if openBrowser {
		go func(url string) {
			time.Sleep(150 * time.Millisecond)
			gui.OpenBrowser(url)
		}("http://" + *addr + "/assets/")
	}
	log.Fatal(server.Serve(ln))
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("failed to encode JSON output: %v", err)
	}
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
