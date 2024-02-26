package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ClientManager struct {
	Clients    map[*Client]bool
	register   chan *Client
	unRegister chan *Client
	broadCast  chan []byte
}

var manager = ClientManager{
	Clients:    map[*Client]bool{},
	register:   make(chan *Client),
	unRegister: make(chan *Client),
	broadCast:  make(chan []byte),
}

type Client struct {
	id   int
	conn *websocket.Conn
	send chan []byte
}

type Message struct {
	Sender  string
	Content string
}

func (manager *ClientManager) start() {
	for {
		select {
		case client := <-manager.register:
			manager.Clients[client] = true

		case client := <-manager.unRegister:
			close(client.send)
			delete(manager.Clients, client)

		case message := <-manager.broadCast:
			for client := range manager.Clients {
				select {
				case client.send <- message:
				default:
					fmt.Println("the client is not active, closing the connection")
					close(client.send)
					delete(manager.Clients, client)
					// client connection to be closed here
				}
			}
		}
	}
}

func (client *Client) read() {
	defer func() {
		manager.unRegister <- client
		client.conn.Close()
	}()
	for {
		_, p, err := client.conn.ReadMessage()

		if err != nil {
			log.Println(err)
			manager.unRegister <- client
			client.conn.Close()
			break
		}

		manager.broadCast <- p
	}
}

func (client *Client) write() {
	defer func() {
		client.conn.Close()
	}()

	for message := range client.send {

		client.conn.WriteMessage(websocket.TextMessage, message)

	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the homepage")
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("client device connected")

	client := &Client{
		id:   1,
		conn: ws,
		send: make(chan []byte),
	}

	manager.register <- client

	go client.read()

	go client.write()

}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("starting main function")
	setupRoutes()

	go manager.start()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
