package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"os"
	"os/signal"
)

type InputConfig struct {
	TelegramAPIKey string `json:"telegram_api_key"`
	TelegramChatID string `json:"telegram_chat_id"`
}

func main() {
	var inputFile = flag.String("input", "", "Path to input json file")
	flag.Parse()
	inputData, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Println("Error reading input file")
	}

	var inputConfig InputConfig
	if err := json.Unmarshal(inputData, &inputConfig); err != nil {
		fmt.Println("Error parsing input json")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(inputConfig.TelegramAPIKey, opts...)
	if err != nil {
		fmt.Println("Error creating bot")
	}

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "I can send messages",
	})
	if err != nil {
		fmt.Println(err)
	}
}
