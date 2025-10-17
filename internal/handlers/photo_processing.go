package handlers

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	chat_gpt "github.com/rinatantsura/telegram-sticker-pack-creator/internal/chat-gpt"
	errors_internal "github.com/rinatantsura/telegram-sticker-pack-creator/internal/errors"
	telegram_api "github.com/rinatantsura/telegram-sticker-pack-creator/internal/telegram-api"
	"os"
)

type Handler struct {
	telegram_api.ClientTelegram
	chat_gpt.ClientChatGPT
}

func (h Handler) Handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || len(update.Message.Photo) == 0 {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please, send me a photo",
		})
		if err != nil {
			errors_internal.ProcessMessage(ctx, b, err, update.Message.Chat.ID)
		}
		return
	}
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
	})
	if err != nil {
		wrappedError := errors_internal.ErrInternalService.Wrap(err)
		errors_internal.ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}
	inputPhotoName, err := h.SavePhoto(file.FilePath, update.Message.Date)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "I got and saved your photo.",
	})
	if err != nil {
		errors_internal.ProcessMessage(ctx, b, err, update.Message.Chat.ID)
	}
	outputPhotoPath, err := h.DeletePhotoBackground(ctx, inputPhotoName)

	if err != nil {
		wrappedError := errors_internal.ErrInternalService.Wrap(err)
		errors_internal.ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}
	openedFile, err := mustOpenFile(outputPhotoPath)
	if err != nil {
		wrappedError := errors_internal.ErrInternalService.Wrap(err)
		errors_internal.ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}

	_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: update.Message.Chat.ID,
		Photo: &models.InputFileUpload{
			Filename: outputPhotoPath,
			Data:     openedFile,
		},
	})
	if err != nil {
		errors_internal.ProcessMessage(ctx, b, err, update.Message.Chat.ID)
	}
	return
}

func mustOpenFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}
