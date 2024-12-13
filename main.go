package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"networking-lab02/pkg/chatroom"
	"networking-lab02/pkg/games"
	"os"
	"regexp"
	"strconv"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan chatroom.Message)

const port = 8080

func main() {
	store := games.NewMemStore()
	gamesHandler := NewGamesHandler(store)

	mux := http.NewServeMux()

	mux.Handle("/", &homeHandler{})
	mux.Handle("/games", gamesHandler)
	mux.Handle("/games/", gamesHandler)
	mux.Handle("/file", &fileUploadHandler{})

	mux.HandleFunc("/websocket", handleConnections)

	go handleMessages()

	fmt.Println("Server started on :" + strconv.Itoa(port))
	err := http.ListenAndServe(":"+strconv.Itoa(port), mux)

	if err != nil {
		panic("Error starting server: " + err.Error())
	}

}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := chatroom.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		var msg chatroom.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println(err)
			delete(clients, conn)
			return
		}

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

type homeHandler struct{}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("This is my home page"))
}

type fileUploadHandler struct{}

func (h *fileUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File not found in request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	err = os.MkdirAll("uploads", os.ModePerm)
	if err != nil {
		fmt.Println("Error creating uploads directory:", err)
		http.Error(w, "Unable to create file on server", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create("uploads/" + header.Filename)
	if err != nil {
		http.Error(w, "Unable to create file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save the file", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("File uploaded successfully!"))
}

func InternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 Internal Server Error"))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Not Found"))
}

// GamesHandler implements http.Handler and dispatches requests to the store
type GamesHandler struct {
	store gameStore
}

func NewGamesHandler(s gameStore) *GamesHandler {
	return &GamesHandler{
		store: s,
	}
}

var (
	GameRe       = regexp.MustCompile(`^/games/*$`)
	GameReWithID = regexp.MustCompile(`^/games/([a-z0-9]+)$`)
)

func (h *GamesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && GameRe.MatchString(r.URL.Path):
		h.CreateGame(w, r)
		return
	case r.Method == http.MethodGet && GameRe.MatchString(r.URL.Path):
		h.ListGames(w, r)
		return
	case r.Method == http.MethodGet && GameReWithID.MatchString(r.URL.Path):
		h.GetGame(w, r)
		return
	case r.Method == http.MethodPut && GameReWithID.MatchString(r.URL.Path):
		h.UpdateGame(w, r)
		return
	case r.Method == http.MethodDelete && GameReWithID.MatchString(r.URL.Path):
		h.DeleteGame(w, r)
		return
	default:
		return
	}
}

func (h *GamesHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	// Game object that will be populated from JSON payload
	var game games.Game
	err := json.NewDecoder(r.Body).Decode(&game)

	if err != nil {
		InternalServerErrorHandler(w, r)
		return
	}
	err = h.store.Add(game)

	// Set the status code to 201
	w.WriteHeader(http.StatusCreated)
}
func (h *GamesHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	// Get specific query parameter by key
	//var page Page = {}
	pageNr, err := strconv.Atoi(params.Get("page"))
	if err != nil {
		fmt.Println("Error: query param invalid!")
		return
	}
	size, err := strconv.Atoi(params.Get("size"))
	if err != nil {
		fmt.Println("Error: query param invalid!")
		return
	}
	page := games.Page{
		Page: int32(pageNr),
		Size: int32(size),
	}

	resources, err := h.store.List(page)

	jsonBytes, err := json.Marshal(resources)
	if err != nil {
		InternalServerErrorHandler(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
func (h *GamesHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	// Extract the resource ID using a regex
	matches := GameReWithID.FindStringSubmatch(r.URL.Path)
	// Expect matches to be length >= 2 (full string + 1 matching group)
	if len(matches) < 2 {
		InternalServerErrorHandler(w, r)
		return
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		fmt.Println("Error:", err)
		InternalServerErrorHandler(w, r)
		return
	}

	// Retrieve recipe from the store
	game, err := h.store.Get(num)
	if err != nil {
		// Special case of NotFound Error
		if err == games.NotFoundErr {
			NotFoundHandler(w, r)
			return
		}

		// Every other error
		InternalServerErrorHandler(w, r)
		return
	}

	// Convert the struct into JSON payload
	jsonBytes, err := json.Marshal(game)
	if err != nil {
		InternalServerErrorHandler(w, r)
		return
	}

	// Write the results
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
func (h *GamesHandler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	matches := GameReWithID.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		InternalServerErrorHandler(w, r)
		return
	}

	// Recipe object that will be populated from JSON payload
	var game games.Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		InternalServerErrorHandler(w, r)
		return
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		fmt.Println("Error:", err)
		InternalServerErrorHandler(w, r)
		return
	}

	if err := h.store.Update(num, game); err != nil {
		if err == games.NotFoundErr {
			NotFoundHandler(w, r)
			return
		}
		InternalServerErrorHandler(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (h *GamesHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	matches := GameReWithID.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		InternalServerErrorHandler(w, r)
		return
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		fmt.Println("Error:", err)
		InternalServerErrorHandler(w, r)
		return
	}

	if err := h.store.Remove(num); err != nil {
		InternalServerErrorHandler(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type gameStore interface {
	Add(game games.Game) error
	Get(id int) (games.Game, error)
	Update(id int, game games.Game) error
	List(page games.Page) ([]games.Game, error)
	Remove(id int) error
}
