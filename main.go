package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID   int
	Name string
}

type Project struct {
	ProjectID         int
	ProjectName       string
	SubscribedUserIDs []int
}

var (
	clients     = make(map[int]chan string) // Store SSE clients per user ID
	clientsLock sync.Mutex
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})

	r.GET("/events", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("id"))
		if err != nil {
			// c.AbortWithError(http.StatusBadRequest, "Invalid user ID")
			return
		}
		fmt.Println("user connected", userID)
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			// c.AbortWithError(http.StatusNotImplemented, "Streaming unsupported")
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		clientChan := make(chan string)
		clientsLock.Lock()
		clients[userID] = clientChan
		clientsLock.Unlock()

		defer func() {
			clientsLock.Lock()
			delete(clients, userID)
			clientsLock.Unlock()
			close(clientChan) // Close the client channel when the user disconnects
		}()

		for {
			select {
			case msg, ok := <-clientChan:
				if !ok {
					// Client channel closed, user disconnected
					return
				}

				newMessage := struct {
					Data string `json:"data"`
				}{
					Data: msg,
				}

				jsonMessage, _ := json.Marshal(newMessage)

				fmt.Fprintf(c.Writer, "data: %s \n\n", jsonMessage)

				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	})

	r.POST("/post", func(c *gin.Context) {
		projectIDParam := c.Query("projectid")
		projectID, err := strconv.Atoi(projectIDParam)
		if err != nil {
			// c.AbortWithError(http.StatusBadRequest, "Invalid project ID")
			return
		}

		message := c.PostForm("message")

		projects := []Project{
			{ProjectID: 1, ProjectName: "fastandfearuos", SubscribedUserIDs: []int{1, 2}},
			{ProjectID: 2, ProjectName: "tokyodrift", SubscribedUserIDs: []int{3, 4}},
		}

		for _, project := range projects {
			if project.ProjectID == projectID {
				for _, userID := range project.SubscribedUserIDs {
					clientsLock.Lock()
					clientChan, exists := clients[userID]
					clientsLock.Unlock()

					if exists {
						clientChan <- message
					}
				}
				break
			}
		}

		c.Status(http.StatusOK)
	})

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			return
		}

		c.Next()
	})

	log.Println("Server started on http://localhost:8080")
	log.Fatal(r.Run(":8080"))
}

// ------------------------------------------------------------
