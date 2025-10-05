package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-telegram/bot"
)

var (
	MsgGetFileFromTelegramChat = Error{CustomerMessage: "Ошибка получения файла фотографии. Попробуйте позже."}
	MsgInternalService         = Error{CustomerMessage: "Внутренняя ошибка сервера, попробуйте еще раз"}
	MsgProcessFile             = Error{CustomerMessage: "Не удалось обработать картинку"}
	MsgSaveFile                = Error{CustomerMessage: "Не удалось сохранить картинку"}
	MsgRequestChatGPT          = Error{CustomerMessage: "Ошибка при отправке запроса. Попробуйте позже."}
	MsgUnknow                  = Error{CustomerMessage: "Неизвестная ошибка, попробуйте еще раз"}
)

var (
	ErrBadStatusCodeTelegram = errors.New("telegram API returned non-200 HTTP status")
	ErrBadStatusCodeChatGPT  = errors.New("chat gpt API returned non-200 HTTP status")
)

type Error struct {
	CustomerMessage string
	ErrInternal     error
}

func (e Error) Error() string {
	return fmt.Sprintf("Interanl Error: %s, Customer Message: %s", e.ErrInternal, e.CustomerMessage)
}

func (e Error) Wrap(err error) error {
	e.ErrInternal = err
	return e
}

func ProcessMessage(ctx context.Context, b *bot.Bot, err error, chatID int64) {
	if err == nil {
		return
	}
	e, ok := err.(Error)
	if ok {
		fmt.Printf("Error: %s\n", e.Error())
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Произошла ошибка:\n %s", e.CustomerMessage),
		})
		if errSendMessage != nil {
			fmt.Printf("Fail to send error message for customer: %s", errSendMessage.Error())
		}
	} else {
		errWrapped := MsgUnknow.Wrap(err)
		ProcessMessage(ctx, b, errWrapped, chatID)
	}
}
