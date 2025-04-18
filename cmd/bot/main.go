package main

import "github.com/vnxcius/sss-backend/internal/infra/bot"

func main() {
	bot.StartBot()
	<-make(chan struct{})
}