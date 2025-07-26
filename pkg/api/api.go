package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type APIMetric struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
}

type APISymptom struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
}

type APIRiskItem struct {
	Service          string       `json:"service"`
	Alert            string       `json:"alert"`
	Severity         string       `json:"severity"`
	Score            int          `json:"score"`
	Symptoms         []APISymptom `json:"symptoms"`
	Metrics          []APIMetric  `json:"metrics"`
	Summary          string       `json:"summary"`
	Risk             string       `json:"risk"`
	Confidence       float64      `json:"confidence"`
	RootCause        string       `json:"root_cause"`
	ImmediateActions []string     `json:"immediate_actions"`
	Investigation    []string     `json:"investigation_steps"`
	Prevention       string       `json:"prevention"`
	Timestamp        string       `json:"timestamp"`
}

type WebSocketMessage struct {
	Type string        `json:"type"`
	Data []APIRiskItem `json:"data"`
}

type WebSocketClient struct {
	conn   *websocket.Conn
	send   chan WebSocketMessage
	hub    *WebSocketHub
}

type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	stop       chan struct{} 
}


var (
	currentAPIRisks []APIRiskItem
	riskMu          sync.RWMutex
	wsHub          *WebSocketHub
	upgrader       = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

var server *http.Server

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		stop:       make(chan struct{}), 
	}
}


func (h *WebSocketHub) Run() {
	for {
		select {
		case <-h.stop: 
			log.Println("🛑 WebSocket hub stopping...")
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("📡 WebSocket client connected (total: %d)", len(h.clients))
			
			// Send current data to new client
			riskMu.RLock()
			currentData := make([]APIRiskItem, len(currentAPIRisks))
			copy(currentData, currentAPIRisks)
			riskMu.RUnlock()
			
			select {
			case client.send <- WebSocketMessage{Type: "risks_update", Data: currentData}:
			default:
				close(client.send)
				delete(h.clients, client)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("📡 WebSocket client disconnected (total: %d)", len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) Stop() {
	close(h.stop)
}

func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established with %s", r.RemoteAddr)
	client := &WebSocketClient{
		conn: conn,
		send: make(chan WebSocketMessage, 256),
		hub:  wsHub,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func StartServer() *http.Server {
	// Initialize WebSocket hub
	wsHub = NewWebSocketHub()
	go wsHub.Run()

	// Create dedicated mux for better control
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", handleWebSocket)
	
	// REST API endpoint
	mux.HandleFunc("/api/risks", func(w http.ResponseWriter, r *http.Request) {
		riskMu.RLock()
		defer riskMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentAPIRisks)
	})

	// Frontend handler
	mux.Handle("/", http.FileServer(http.Dir("./dashboard/dist")))

	server = &http.Server{
		Addr:    ":8090",
		Handler: mux,
	}
	
	fmt.Println("🚀 API server running at: http://localhost:8090")
	fmt.Println("   - Dashboard: http://localhost:8090")
	fmt.Println("   - WebSocket: ws://localhost:8090/ws") 
	fmt.Println("   - REST API:  http://localhost:8090/api/risks")
	go server.ListenAndServe()
	return server
}

func StopServer() {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		fmt.Println("🛑 API server stopped")
	}

	if wsHub != nil {
		wsHub.Stop()
	}
}

func UpdateRisks(newRisks []APIRiskItem) {
	riskMu.Lock()
	currentAPIRisks = newRisks
	riskMu.Unlock()

	// Broadcast update to all WebSocket clients
	if wsHub != nil {
		select {
		case wsHub.broadcast <- WebSocketMessage{Type: "risks_update", Data: newRisks}:
		default:
			log.Printf("WebSocket broadcast channel full, skipping update")
		}
	}
}
