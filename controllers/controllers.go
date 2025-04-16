package controllers

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

func VerifyToken(c *gin.Context) {
	validToken := os.Getenv("TOKEN")
	var req VerifyTokenRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if validToken == req.Token {
		c.SetCookie(
			"sss-token",
			validToken,
			30*24*60*60, // 30 days
			"/",
			"",
			false,
			true,
		)

		c.JSON(http.StatusOK, gin.H{"message": "Token verificado com sucesso"})

		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
}

func StartServer(c *gin.Context) {
	log.Println("Starting server...")
}

func StopServer(c *gin.Context) {
	log.Println("Stopping server...")
}

func RestartServer(c *gin.Context) {
	log.Println("Restarting server...")
}
