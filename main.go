package main

import (
	"backend/auth-service"
	"backend/chat-service"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
	r := gin.Default()
	r.POST("/auth/reg", auth.RegHandler)
	r.POST("/auth/auth", auth.AuthHandler)
	r.POST("/chats/room_create", chats.RoomCreateHandler)
	r.POST("/chats/room_sign", chats.RoomSignHandler)
	r.POST("/chats/manage", chats.ExitRoomHandler)
	r.POST("/chats/send", chats.SendHandler)
	r.GET("/connect", chats.ConnectHandler)
	auth.InitAuthDB()
	chats.InitChatDB()
	err := r.Run(port)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("running")
}
