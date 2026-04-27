package chats

import (
	"github.com/gorilla/websocket"
	"net/http"
	"backend/auth-service"
	"github.com/gin-gonic/gin"
	"log"
)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func RoomCreateHandler(c *gin.Context) {
	var req CreateRoomRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = repostitoryCreateDB(req.RoomName, req.RoomType, req.AccessType, parsed.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully"})
}

func RoomSignHandler(c *gin.Context) {
	var req SignRoomRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err, ok := manageRoomDB(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot sign into room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully"})
}

func ExitRoomHandler(c *gin.Context) {
	var req SignRoomRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err, ok := manageRoomDB(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot leave room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully"})
}

func SendHandler(c *gin.Context) {
	var req SendReq

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err,_ = sendService(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully"})
}

func ConnectHandler(c *gin.Context) {
token := c.GetHeader("JWT")
	_, err := auth.ParseJWT(token)
	if err != nil {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	log.Println("client connected")
go Reader(conn)
return
}
