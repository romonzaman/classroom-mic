package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	ID      string
	Name    string
	Role    string
	Channel string
	Conn    *websocket.Conn
}

type Hub struct {
	mu       sync.Mutex
	teacher  *Client
	students map[string]*Client
}

func NewHub() *Hub {
	return &Hub{students: make(map[string]*Client)}
}

func getTurnConfig() map[string]interface{} {
	return map[string]interface{}{
		"urls":       os.Getenv("TURN_URL"),
		"username":   os.Getenv("TURN_USERNAME"),
		"credential": os.Getenv("TURN_CREDENTIAL"),
	}
}

func (h *Hub) AddClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Role == "teacher" {
		h.teacher = c
		log.Println("Teacher connected:", c.Name)
		// Send current connected students to teacher
		h.sendConnectedStudentsToTeacher()
	} else {
		h.students[c.ID] = c
		log.Println("Student connected:", c.ID)
		// Notify teacher about new student
		h.notifyTeacherStudentUpdate(c, "connected")
	}
}

func (h *Hub) RemoveClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Role == "teacher" {
		h.teacher = nil
		log.Println("Teacher disconnected")
	} else {
		studentID := c.ID
		delete(h.students, c.ID)
		log.Println("Student disconnected:", studentID)
		// Notify teacher about student disconnection
		h.notifyTeacherStudentUpdate(c, "disconnected")
		// Also remove from raised hands if they were there
		removePayload := map[string]interface{}{"type": "remove_from_raised_hands", "id": studentID}
		if h.teacher != nil {
			h.teacher.Conn.WriteJSON(removePayload)
		}
	}
}

func (h *Hub) SendToTeacher(msg interface{}) error {
	h.mu.Lock()
	t := h.teacher
	h.mu.Unlock()
	if t == nil {
		return fmt.Errorf("no teacher connected")
	}
	return t.Conn.WriteJSON(msg)
}

func (h *Hub) SendToStudent(id string, msg interface{}) error {
	h.mu.Lock()
	s := h.students[id]
	h.mu.Unlock()
	if s == nil {
		return fmt.Errorf("student not found")
	}
	return s.Conn.WriteJSON(msg)
}

func (h *Hub) sendConnectedStudentsToTeacher() {
	if h.teacher == nil {
		return
	}
	
	studentsList := make([]map[string]interface{}, 0, len(h.students))
	for _, student := range h.students {
		studentsList = append(studentsList, map[string]interface{}{
			"id":      student.ID,
			"name":    student.Name,
			"channel": student.Channel,
		})
	}
	
	msg := map[string]interface{}{
		"type":     "connected_students",
		"students": studentsList,
	}
	
	h.teacher.Conn.WriteJSON(msg)
}

func (h *Hub) notifyTeacherStudentUpdate(c *Client, action string) {
	if h.teacher == nil {
		return
	}
	
	msg := map[string]interface{}{
		"type":   "student_update",
		"action": action,
		"id":     c.ID,
		"name":   c.Name,
		"channel": c.Channel,
	}
	
	h.teacher.Conn.WriteJSON(msg)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	e := echo.New()
	hub := NewHub()

	e.Static("/", "static")

	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		id := c.QueryParam("id")
		name := c.QueryParam("name")
		role := c.QueryParam("role")
		channel := c.QueryParam("channel")
		if role == "" {
			role = "student"
		}

		client := &Client{ID: id, Name: name, Role: role, Channel: channel, Conn: conn}
		hub.AddClient(client)

		// Send TURN server configuration to client
		turnConfig := getTurnConfig()
		configMsg := map[string]interface{}{
			"type": "config",
			"turn": turnConfig,
		}
		conn.WriteJSON(configMsg)

		defer func() {
			hub.RemoveClient(client)
			conn.Close()
		}()

		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("read error:", err)
				break
			}

			t, _ := msg["type"].(string)
		switch t {
		case "raise_hand":
			payload := map[string]interface{}{"type": "raise_hand", "id": client.ID, "name": client.Name, "channel": client.Channel}
			_ = hub.SendToTeacher(payload)
			case "offer":
				payload := map[string]interface{}{"type": "offer", "from": client.ID, "sdp": msg["sdp"]}
				_ = hub.SendToTeacher(payload)
			case "answer":
				toID, _ := msg["to"].(string)
				payload := map[string]interface{}{"type": "answer", "sdp": msg["sdp"]}
				_ = hub.SendToStudent(toID, payload)
			case "ice":
				target, _ := msg["to"].(string)
				payload := map[string]interface{}{"type": "ice", "candidate": msg["candidate"], "from": client.ID}
				if target == "teacher" {
					_ = hub.SendToTeacher(payload)
				} else {
					_ = hub.SendToStudent(target, payload)
				}
			case "allow":
				toID, _ := msg["to"].(string)
				payload := map[string]interface{}{"type": "allowed"}
				_ = hub.SendToStudent(toID, payload)
		case "mute":
			toID, _ := msg["to"].(string)
			payload := map[string]interface{}{"type": "mute"}
			_ = hub.SendToStudent(toID, payload)
			// Notify teacher to remove from raised hands list
			removePayload := map[string]interface{}{"type": "remove_from_raised_hands", "id": toID}
			_ = hub.SendToTeacher(removePayload)
			default:
				log.Println("unknown message type:", t)
			}
		}

		return nil
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}
	
	log.Printf("Starting server on :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
