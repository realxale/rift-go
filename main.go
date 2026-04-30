package main

import (
	"backend/auth-service"
	"backend/chat-service"
	"backend/database"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
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

	// Инициализация БД (все таблицы)
	database.InitAllTables()

	chats.InitLimit()
	err := r.Run(port)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("running")
}