package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/go-telegram/bot"
	chat_gpt "github.com/rinatantsura/telegram-sticker-pack-creator/internal/chat-gpt"
	"github.com/rinatantsura/telegram-sticker-pack-creator/internal/handlers"
	telegram_api "github.com/rinatantsura/telegram-sticker-pack-creator/internal/telegram-api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

type InputConfig struct {
	TelegramAPIKey      string `json:"telegram_api_key"`
	TelegramChatID      string `json:"telegram_chat_id"`
	ChatGPTAPIKey       string `json:"chat_gpt_api_key"`
	TelegramFileBaseURL string `json:"telegram_file_base_url"`
	ChatGPTBaseURL      string `json:"chat_gpt_base_url"`
}

func main() {
	log.Logger = log.Level(zerolog.InfoLevel)
	var inputFile = flag.String("input", "", "Path to input json file")
	flag.Parse()
	inputData, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatal().Err(err).Str("file", *inputFile).Msg("Failed to read input file")
	}
	var inputConfig InputConfig
	if err = json.Unmarshal(inputData, &inputConfig); err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	clientTelegram := telegram_api.NewClient(inputConfig.TelegramFileBaseURL, inputConfig.TelegramAPIKey)
	clientChatGPT := chat_gpt.NewClient(inputConfig.ChatGPTAPIKey, inputConfig.ChatGPTBaseURL)

	h := handlers.Handler{
		ClientTelegram: telegram_api.ClientTelegram{TelegramFileBaseURL: clientTelegram.TelegramFileBaseURL, Token: clientTelegram.Token},
		ClientChatGPT:  chat_gpt.ClientChatGPT{APIKey: clientChatGPT.APIKey, BaseURL: clientChatGPT.BaseURL},
	}
	opts := []bot.Option{
		bot.WithDefaultHandler(h.Handler),
	}

	b, err := bot.New(inputConfig.TelegramAPIKey, opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Telegram bot")
	}
	b.Start(ctx)
}
