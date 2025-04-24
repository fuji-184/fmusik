package controller

import (
    "net/http"
    "log"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader {
    CheckOrigin: func (r *http.Request) bool {
        return true
    },
}

var rooms = make(map[string][]*websocket.Conn)

func WsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("error upgrading to websocket: %v", err)
    }
    defer conn.Close()

    room := r.URL.Query().Get("room")
    if room == "" {
        room = "default"
    }

    rooms[room] = append(rooms[room], conn)

    defer func() {
        conn.Close()
        removeFromRoom(room, conn)
    }()

    for {
        msg_type, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println("error reading ws messae: %v", err)
        }

        for _, client := range rooms[room] {
            if err := client.WriteMessage(msg_type, msg); err != nil {
                log.Println("error writing ws message: %v", err)
            }
        }
    }
}

func removeFromRoom(room string, conn *websocket.Conn) {
    conns := rooms[room]
    for i, client := range conns {
        if client == conn {
            rooms[room] = append(conns[:i], conns[i+1:]...)
            break
        }
    }

    if len(rooms[room]) == 0 {
        delete(rooms, room)
    }
}

