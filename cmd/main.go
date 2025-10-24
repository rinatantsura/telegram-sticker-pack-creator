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
	"strings"
)

type InputConfig struct {
	TelegramAPIKey      string `json:"telegram_api_key"`
	TelegramChatID      string `json:"telegram_chat_id"`
	ChatGPTAPIKey       string `json:"chat_gpt_api_key"`
	TelegramFileBaseURL string `json:"telegram_file_base_url"`
	ChatGPTBaseURL      string `json:"chat_gpt_base_url"`
	LogLevel            string `json:"log_level"`
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("Starting application with default log level INFO")

	var inputFile = flag.String("input", "", "Path to input json file")
	flag.Parse()
	inputConfig, err := readConfig(*inputFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load input config")
	}

	setupLogger(inputConfig.LogLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	clientTelegram := telegram_api.NewClient(inputConfig.TelegramFileBaseURL, inputConfig.TelegramAPIKey)
	clientChatGPT := chat_gpt.NewClient(inputConfig.ChatGPTAPIKey, inputConfig.ChatGPTBaseURL)

	log.Info().
		Fields(map[string]interface{}{
			"telegram_base_url": clientTelegram.TelegramFileBaseURL,
			"chatgpt_base_url":  clientChatGPT.BaseURL,
		}).
		Msg("Clients initialized")

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
	log.Info().Msg("Telegram bot successfully initialized")

	b.Start(ctx)

	log.Info().Msg("Bot started and listening for updates")
}

func readConfig(inputFile string) (*InputConfig, error) {
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatal().Err(err).Str("file", inputFile).Msg("Failed to read input file")
	}
	var inputConfig InputConfig
	if err = json.Unmarshal(inputData, &inputConfig); err != nil {
		log.Fatal().Err(err).Str("file", inputFile).Msg("Failed to parse JSON config")
	}
	return &inputConfig, nil
}

func setupLogger(logLevel string) {
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = zerolog.InfoLevel
		log.Warn().Str("provided_level", logLevel).Msg("Invalid log level, fallback to INFO")
	}
	zerolog.SetGlobalLevel(level)
	log.Info().Str("level", level.String()).Msg("Log level configured")
}
