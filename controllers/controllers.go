package controllers

import (
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
)

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

var cmd *exec.Cmd

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
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}

	cmd = exec.Command("bash", "/home/simon/minecraft/start.sh")
}
