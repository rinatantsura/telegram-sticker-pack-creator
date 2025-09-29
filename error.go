package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-telegram/bot"
)

var (
	MsgGetFileFromTelegramChat = Error{CustomerMessage: "Ошибка получения файла фотографии"}
	MsgInternalService         = Error{CustomerMessage: "Внутренняя ошибка сервера, попробуйте еще раз"}
	MsgProcessFile             = Error{CustomerMessage: "Не удалось обработать картинку"}
	MsgSaveFile                = Error{CustomerMessage: "Не удалось сохранить картинку"}
	MsgRequestChatGPT          = Error{CustomerMessage: "Ошибка при отправке запроса. Попробуйте позже."}
	MsgMultipartCreate         = Error{CustomerMessage: "Ошибка при подготовке файла для отправки. Попробуйте позже."}
	MsgInternalFileErr         = Error{CustomerMessage: "Внутренняя ошибка работы с файлом."}
	MsgJSONDecode              = Error{CustomerMessage: "Не удалось обработать ответ сервера. Попробуйте позже."}
	MsgNoImageReturned         = Error{CustomerMessage: "Сервер не вернул изображение. Попробуйте другой запрос."}
	MsgUnknow                  = Error{CustomerMessage: "Неизвестная ошибка, попробуйте еще раз"}
)

var (
	ErrBadStatusCodeTelegram = errors.New("telegram API returned non-200 HTTP status")
	ErrProcessFile           = errors.New("failed to open file")
	ErrMultipartCreatePart   = errors.New("failed to create multipart part")
	ErrMultipartWriteFile    = errors.New("failed to write file to multipart part")
	ErrCloseFile             = errors.New("failed to close file")
	ErrCreateRequestChatGPT  = errors.New("failed to create request to chat gpt")
	ErrSendRequestChatGPT    = errors.New("failed to send request to chat gpt")
	ErrBadStatusCodeChatGPT  = errors.New("chat gpt API returned non-200 HTTP status")
	ErrJSONDecode            = errors.New("failed to decode JSON response")
	ErrNoImageReturned       = errors.New("no image returned in response")
	ErrBase64Decode          = errors.New("failed to decode base64 image data")
	ErrWriteFile             = errors.New("failed to write image to file")
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

//func (e Error) Unwrap() error {
//	return e.ErrInternal
//}

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
