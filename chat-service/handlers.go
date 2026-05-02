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
	err = repostitoryCreateDB(req.RoomName, string(req.RoomType), string(req.AccessType), parsed.Username)
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

	err = ManageRoomService(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	err = ManageRoomService(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func RoomsListHandler(c *gin.Context) {
	var req struct {
		JWT string `json:"jwt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rooms, err := roomsListService(req.JWT)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

func ConnectHandler(c *gin.Context) {
	token := c.GetHeader("JWT")
	if token == "" {
		token = c.Query("jwt")
	}
	_, err := auth.ParseJWT(token)
	if err != nil {
		log.Println("ws auth error:", err)
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