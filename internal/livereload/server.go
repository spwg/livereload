package livereload

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

type ReloadHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewReloadHub() *ReloadHub {
	return &ReloadHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *ReloadHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (app *Livereload) serveScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(app.LivereloadJS)
}

func (app *Livereload) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if app.Log != nil {
			app.Log.Println("upgrade:", err)
		} else {
			log.Println("upgrade:", err)
		}
		return
	}
	app.Hub.register <- conn

	// Keep connection alive
	go func() {
		defer func() {
			app.Hub.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (app *Livereload) StartServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/livereload.js", app.serveScript)
	mux.HandleFunc("/ws", app.serveWs)

	addr := fmt.Sprintf("%s:%d", app.ReloadHost, app.ReloadPort)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	app.Log.Printf("Livereload server listening on http://%s", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	go app.Hub.Run()
}
