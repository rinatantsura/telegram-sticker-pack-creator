package errors

import (
	"errors"
)

var (
	ErrInternalService = Error{CustomerMessage: "Internal server error, please try again."}
	ErrUnknow          = Error{CustomerMessage: "Неизвестная ошибка, попробуйте еще раз"}
)

var (
	ErrBadStatusCodeTelegram = errors.New("telegram API returned non-200 HTTP status")
	ErrBadStatusCodeChatGPT  = errors.New("chat gpt API returned non-200 HTTP status")
)
