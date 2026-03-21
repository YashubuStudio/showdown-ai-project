package gui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/ysu/showdown-go-client/internal/httpapi"
)

//go:embed assets/*
var assets embed.FS

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	api := httpapi.New()
	assetsFS, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	mux.Handle("/api/", api.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/assets/", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	})
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))
	return mux
}

func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("gui: failed to open browser: %v", err)
	}
}
