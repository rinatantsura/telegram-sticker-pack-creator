package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"io"
	"net/http"
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

	if update.Message == nil || len(update.Message.Photo) == 0 {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please, send me a photo",
		})
		if err != nil {
			fmt.Println("Error sending message")
		}
		return
	}
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
	})
	if err != nil {
		fmt.Println("Error getting file info")
	}

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Token(), file.FilePath)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading file")
		return
	}
	defer resp.Body.Close()

	filename := fmt.Sprintf("photo_%d.jpg", update.Message.Date)
	out, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file")
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error copying file")
		return
	}

	fmt.Println("Saved photo", filename)
}
