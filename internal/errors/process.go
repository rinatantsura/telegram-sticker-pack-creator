package errors

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
)

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
			fmt.Printf("Fail to send errx message for customer: %s", errSendMessage.Error())
		}
	} else {
		errWrapped := ErrUnknow.Wrap(err)
		ProcessMessage(ctx, b, errWrapped, chatID)
	}
}
