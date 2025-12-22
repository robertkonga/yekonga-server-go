package Yekonga

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/robertkonga/yekonga-server/datatype"
	"github.com/robertkonga/yekonga-server/helper"
	"github.com/robertkonga/yekonga-server/helper/console"

	// "github.com/robertkonga/yekonga-server/plugins/socketio"
	// "github.com/robertkonga/yekonga-server/plugins/socketio/engineio"
	// "github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport"
	// "github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport/polling"
	// "github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport/websocket"

	"github.com/robertkonga/yekonga-server/plugins/websocket"
)

// Easier to get running with CORS. Thanks for help @Vindexus and @erkie
var allowOriginFunc = func(r *http.Request) bool {
	return true
}

func NewSocketServer(y *YekongaData) *SocketServer {
	s := &SocketServer{
		namespaces: make(map[string]*Namespace),
		app:        y,
	}

	root := s.Of("/") // Root namespace "/"

	root.OnConnect(func(c *Client) {
		console.Log("New client connected: %s (namespace: /)", c.id)

		// c.On("chat_message", func(data interface{}) {
		// 	console.Log("Received === chat_message from %s: %s", c.id, data)
		// })

		c.EmitToClient("message", "Welcome to the Go WebSocket server!")
		c.EmitToClient("message", datatype.DataMap{"message": "robert"})
	})

	root.On("chat_message", func(c *Client, data interface{}) {
		console.Log("Received === chat_message from %s: %s", c.id, data)
	})

	root.OnError(func(c *Client, err error) {
		console.Log("Error on client %s: %v", c.id, err)
	})

	root.OnDisconnect(func(c *Client, reason string) {
		console.Log("Client disconnected: %s (reason: %s)", c.id, reason)
	})

	return s
}

// Namespace represents an isolated group (like Socket.IO namespace)
type Namespace struct {
	mu      sync.Mutex
	clients map[string]*Client
	rooms   map[string]map[string]bool
	app     *YekongaData

	// Socket.IO-style event handlers for this namespace
	onConnect    func(c *Client)
	onDisconnect func(c *Client, reason string)
	onError      func(c *Client, err error)

	// Custom event handlers: event name -> handler function
	eventHandlers map[string]func(c *Client, data interface{})
}

// Client represents a connected WebSocket client
type Client struct {
	id            string
	namespace     *Namespace
	conn          *websocket.Conn
	send          chan []byte
	rooms         map[string]bool // rooms this client is in (for quick lookup)
	eventHandlers map[string]func(data interface{})
}

// Message incoming from client
type Message struct {
	Type  string          `json:"type"`
	MsgID *string         `json:"msgId,omitempty"`
	Data  json.RawMessage `json:"data"`
}

// EventMessage outgoing event
type EventMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// AckMessage acknowledgment response
type AckMessage struct {
	Type    string      `json:"type"`
	MsgID   string      `json:"msgId"`
	Payload interface{} `json:"payload,omitempty"`
}

// SocketServer manages multiple namespaces
type SocketServer struct {
	mu         sync.Mutex
	namespaces map[string]*Namespace
	app        *YekongaData
}

// of returns or creates a namespace
func (s *SocketServer) Of(path string) *Namespace {
	return s.getOrCreateNamespace(path)
}

// of returns or creates a namespace
func (s *SocketServer) Close() {

}

// getOrCreateNamespace returns or creates a namespace
func (s *SocketServer) getOrCreateNamespace(path string) *Namespace {
	if path == "" {
		path = "/"
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if n, ok := s.namespaces[path]; ok {
		return n
	}

	n := &Namespace{
		app:           s.app,
		clients:       make(map[string]*Client),
		rooms:         make(map[string]map[string]bool),
		eventHandlers: make(map[string]func(c *Client, data interface{})),
	}
	s.namespaces[path] = n
	return n
}

// WebSocket handler
func (s *SocketServer) ServeWS(w http.ResponseWriter, r *http.Request) {
	nsPath := r.URL.Query().Get("ns")
	if nsPath == "" {
		nsPath = "/"
	}

	namespace := s.getOrCreateNamespace(nsPath)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	clientID := helper.UUID()
	client := &Client{
		id:            clientID,
		namespace:     namespace,
		conn:          conn,
		send:          make(chan []byte, 256),
		rooms:         make(map[string]bool),
		eventHandlers: make(map[string]func(data interface{})),
	}

	namespace.addClient(client)

	// Send client ID (compatible with previous client code)
	idMsg, _ := json.Marshal(EventMessage{Event: "id", Data: json.RawMessage(`"` + clientID + `"`)})
	client.send <- idMsg

	go client.writePump()
	client.readPump()
}

// Socket.IO-style registration methods (applied to a namespace)

func (n *Namespace) OnConnect(handler func(c *Client)) {
	n.onConnect = handler
}

func (n *Namespace) OnDisconnect(handler func(c *Client, reason string)) {
	n.onDisconnect = handler
}

func (n *Namespace) OnError(handler func(c *Client, err error)) {
	n.onError = handler
}

func (n *Namespace) On(event string, handler func(c *Client, data interface{})) {
	n.eventHandlers[event] = handler
}

func (n *Namespace) OnEvent(event string, handler func(c *Client, data interface{})) {
	n.eventHandlers[event] = handler
}

// Internal methods (same as before, slightly refactored)

func (n *Namespace) addClient(c *Client) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.clients[c.id] = c
	if n.onConnect != nil {
		n.onConnect(c)
	}
}

func (n *Namespace) removeClient(c *Client, reason string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.clients, c.id)

	// Remove from all rooms
	for room := range c.rooms {
		if roomClients, ok := n.rooms[room]; ok {
			delete(roomClients, c.id)
			if len(roomClients) == 0 {
				delete(n.rooms, room)
			}
		}
	}
	c.rooms = nil

	if n.onDisconnect != nil {
		n.onDisconnect(c, reason)
	}
}

func (n *Namespace) JoinRoom(room string, c *Client) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, ok := n.rooms[room]; !ok {
		n.rooms[room] = make(map[string]bool)
	}
	n.rooms[room][c.id] = true
	c.rooms[room] = true
}

func (n *Namespace) LeaveRoom(room string, c *Client) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if roomClients, ok := n.rooms[room]; ok {
		delete(roomClients, c.id)
		if len(roomClients) == 0 {
			delete(n.rooms, room)
		}
	}
	delete(c.rooms, room)
}

func (n *Namespace) EmitToClient(c *Client, event string, data interface{}) {
	msg, err := json.Marshal(EventMessage{Event: event, Data: helper.ToByte((data))})
	if err != nil {
		return
	}
	select {
	case c.send <- msg:
	default:
	}
}

func (n *Namespace) SendToRoom(room, event string, data interface{}, exclude *Client) {
	n.mu.Lock()
	clientsInRoom := make([]*Client, 0)
	if roomClients, ok := n.rooms[room]; ok {
		for cid := range roomClients {
			if client, exists := n.clients[cid]; exists && client != exclude {
				clientsInRoom = append(clientsInRoom, client)
			}
		}
	}
	n.mu.Unlock()

	for _, client := range clientsInRoom {
		n.EmitToClient(client, event, data)
	}
}

func (n *Namespace) Emit(event string, data interface{}, exclude *Client) {
	n.Broadcast(event, data, exclude)
}

func (n *Namespace) Broadcast(event string, data interface{}, exclude *Client) {
	n.mu.Lock()
	allClients := make([]*Client, 0, len(n.clients))
	for _, client := range n.clients {
		if client != exclude {
			allClients = append(allClients, client)
		}
	}
	n.mu.Unlock()

	for _, client := range allClients {
		n.EmitToClient(client, event, data)
	}
}

func (c *Client) On(event string, handler func(data interface{})) {
	c.eventHandlers[event] = handler
}

func (c *Client) OnEvent(event string, handler func(data interface{})) {
	c.eventHandlers[event] = handler
}

func (c *Client) Emit(event string, data interface{}) {
	c.namespace.Emit(event, data, c)
}

func (c *Client) Broadcast(event string, data interface{}) {
	c.namespace.Broadcast(event, data, c)
}

func (c *Client) SendToRoom(room string, event string, data interface{}) {
	c.namespace.SendToRoom(room, event, data, c)
}

func (c *Client) EmitToClient(event string, data interface{}) {
	c.namespace.EmitToClient(c, event, data)
}

func (c *Client) SendBack(event string, data interface{}) {
	c.namespace.EmitToClient(c, event, data)
}

// readPump + message handling
func (c *Client) readPump() {
	defer func() {
		c.namespace.removeClient(c, "closed")
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			reason := "error"
			if websocket.IsCloseError(err) {
				reason = "closed"
			}
			c.namespace.removeClient(c, reason)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *Client) handleMessage(msg *Message) {
	n := c.namespace
	var ackPayload interface{} = "ok"
	var err error

	// First, check if it's a custom event registered via OnEvent
	if msg.Type == "emit" {
		var d struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(msg.Data, &d) == nil && d.Event != "" {
			payload, _ := helper.ToInterface(d.Payload)

			if handler, ok := c.eventHandlers[d.Event]; ok {
				handler(payload)

				// Custom handlers are responsible for replying if needed
				if msg.MsgID != nil {
					ackMsg, _ := json.Marshal(AckMessage{Type: "ack", MsgID: *msg.MsgID, Payload: "ok"})
					c.send <- ackMsg
				}
				return
			}

			if handler, ok := n.eventHandlers[d.Event]; ok {
				handler(c, payload)

				// Custom handlers are responsible for replying if needed
				if msg.MsgID != nil {
					ackMsg, _ := json.Marshal(AckMessage{Type: "ack", MsgID: *msg.MsgID, Payload: "ok"})
					c.send <- ackMsg
				}
				return
			}
		}
	}

	// Built-in protocol handling
	switch msg.Type {
	case "join":
		var d struct {
			Room string `json:"room"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			n.JoinRoom(d.Room, c)
		}
	case "leave":
		var d struct {
			Room string `json:"room"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			n.LeaveRoom(d.Room, c)
		}
	case "rejoinRooms":
		var d struct {
			Rooms []string `json:"rooms"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			for _, room := range d.Rooms {
				n.JoinRoom(room, c)
			}
		}
	case "createChannel":
		// No-op, rooms created on demand
	case "emit", "broadcast":
		var d struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			n.Broadcast(d.Event, d.Payload, c)
		}
	case "toRoom":
		var d struct {
			Room    string          `json:"room"`
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			n.SendToRoom(d.Room, d.Event, d.Payload, c)
		}
	case "toClient":
		var d struct {
			ClientID string          `json:"clientId"`
			Event    string          `json:"event"`
			Payload  json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(msg.Data, &d) == nil {
			n.mu.Lock()
			target, ok := n.clients[d.ClientID]
			n.mu.Unlock()
			if ok {
				n.EmitToClient(target, d.Event, d.Payload)
			}
		}
	}

	if err != nil {
		ackPayload = err.Error()
	}

	if msg.MsgID != nil {
		ackMsg, _ := json.Marshal(AckMessage{
			Type:    "ack",
			MsgID:   *msg.MsgID,
			Payload: ackPayload,
		})
		c.send <- ackMsg
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// func main() {
// 	var addr = flag.String("addr", ":8080", "http service address")
// 	flag.Parse()

// 	server := NewSocketServer()

// 	http.HandleFunc("/ws", server.serveWS)

// 	log.Printf("WebSocket server starting on %s", *addr)
// 	log.Printf("Connect with: ws://localhost%s/ws?ns=/         (root namespace)", *addr)
// 	log.Printf("           or: ws://localhost%s/ws?ns=/chat     (custom namespace)", *addr)
// 	log.Fatal(http.ListenAndServe(*addr, nil))
// }
