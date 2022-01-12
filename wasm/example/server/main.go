package main

import (
	"context"
	"errors"
	"github.com/vmware/transport-go/wasm/example/server/render"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

//go:generate bash -c "GOARCH=wasm; GOOS=js; go build -o render/templates/main.wasm ../sample_wasm_app/main.go"

func main() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL)
	server := exampleServer()

	go func() {
		<-sigChan
		server.Shutdown(context.Background())
	}()

	log.Println("example server running at http://localhost:30080")
	_ = server.ListenAndServe()
}

func exampleServer() *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.RequestURI[1:]) > 0 {
			tmpl := render.GetTemplate(render.DefaultRoute)
			w.WriteHeader(404)
			_ = tmpl.Execute(w, nil)
			return
		}
		tmpl := render.GetTemplate(render.IndexRoute)
		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	handler.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		split := strings.Split(r.RequestURI[1:], "/")
		resourcePath := path.Join("templates", strings.Join(split[1:], "/"))
		b, err := fs.ReadFile(render.FS, resourcePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Write(b)
	})

	s := &http.Server{
		Addr:    ":30080",
		Handler: handler,
	}

	return s
}
