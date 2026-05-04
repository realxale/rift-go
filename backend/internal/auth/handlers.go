package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegHandler(c *gin.Context) {
	var req RegRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = buisnessReg(req)
	if err != nil {
		log.Println("registration error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return

	}

	c.JSON(http.StatusCreated, gin.H{"message": "successfully"})
}
func AuthHandler(c *gin.Context) {
	var req RegRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err, res, jwt := buisnessAuth(&req)
	if err != nil {
		log.Println("auth error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}
	if res == true {
		c.JSON(200, gin.H{
			"token": jwt,
		})
		return
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
	return
}
