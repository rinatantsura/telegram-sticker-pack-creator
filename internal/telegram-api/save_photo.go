package telegram_api

import (
	"fmt"
	"github.com/rinatantsura/telegram-sticker-pack-creator/internal/errors"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
)

type ClientTelegram struct {
	TelegramFileBaseURL string
	Token               string
}

func NewClient(baseUrl string, token string) ClientTelegram {
	return ClientTelegram{
		TelegramFileBaseURL: baseUrl,
		Token:               token,
	}
}

const baseNameOfInputPhoto = "photo_%d.jpg"

func (c ClientTelegram) SavePhoto(filePath string, datePhotoSaving int) (string, error) {
	url := fmt.Sprintf(c.TelegramFileBaseURL, c.Token, filePath)
	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make GET request")
		return "", err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close file")
			return
		}
	}()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status: %v,%s", errors.ErrBadStatusCodeTelegram, resp.Status, string(body))
	}
	log.Info().Int("status_code", resp.StatusCode).Msg("Request to Telegram API succeeded")

	inputPhotoName := fmt.Sprintf(baseNameOfInputPhoto, datePhotoSaving)
	out, err := os.Create(inputPhotoName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file")
		return "", err
	}
	defer func() {
		if err = out.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close file")
			return
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to copy file")
		return "", err
	}
	return inputPhotoName, nil
}
