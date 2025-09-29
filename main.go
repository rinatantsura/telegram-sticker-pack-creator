package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
)

const baseNameOfInputPhoto = "photo_%d.jpg"

type InputConfig struct {
	TelegramAPIKey      string `json:"telegram_api_key"`
	TelegramChatID      string `json:"telegram_chat_id"`
	ChatGPTAPIKey       string `json:"chat_gpt_api_key"`
	TelegramFileBaseURL string `json:"telegram_file_base_url"`
	ChatGPTBaseURL      string `json:"chat_gpt_base_url"`
}

type Handler struct {
	ChatGPTClient
	TelegramClient
}

type ChatGPTClient struct {
	ChatGPTAPIKey  string `json:"chat_gpt_api_key"`
	ChatGPTBaseURL string `json:"chat_gpt_base_url"`
}

type TelegramClient struct {
	TelegramFileBaseURL string `json:"telegram_file_base_url"`
}

func (h Handler) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || len(update.Message.Photo) == 0 {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please, send me a photo",
		})
		if err != nil {
			ProcessMessage(ctx, b, err, update.Message.Chat.ID)
		}
		return
	}
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
	})
	if err != nil {
		wrappedError := MsgGetFileFromTelegramChat.Wrap(err)
		ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}

	url := fmt.Sprintf(h.TelegramFileBaseURL, b.Token(), file.FilePath)
	resp, err := http.Get(url)
	if err != nil {
		wrappedError := MsgInternalService.Wrap(err)
		ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			wrappedError := MsgInternalService.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		}
	}()
	if resp.StatusCode != 200 {
		wrappedError := MsgInternalService.Wrap(ErrBadStatusCodeTelegram)
		ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}

	inputPhotoName := fmt.Sprintf(baseNameOfInputPhoto, update.Message.Date)
	out, err := os.Create(inputPhotoName)
	if err != nil {
		wrappedError := MsgProcessFile.Wrap(err)
		ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}
	defer func() {
		if err = out.Close(); err != nil {
			wrappedError := MsgProcessFile.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		wrappedError := MsgSaveFile.Wrap(err)
		ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "I got and saved your photo.",
	})
	if err != nil {
		ProcessMessage(ctx, b, err, update.Message.Chat.ID)
	}

	err = photoProcessingChatGPT(h.ChatGPTAPIKey, inputPhotoName, h.ChatGPTBaseURL)

	if err != nil {
		switch {
		case errors.Is(err, ErrProcessFile):
			wrappedError := MsgProcessFile.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrMultipartCreatePart), errors.Is(err, ErrMultipartCreatePart):
			wrappedError := MsgMultipartCreate.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrCloseFile):
			wrappedError := MsgInternalFileErr.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrCreateRequestChatGPT), errors.Is(err, ErrSendRequestChatGPT), errors.Is(err, ErrBadStatusCodeChatGPT):
			wrappedError := MsgRequestChatGPT.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrJSONDecode):
			wrappedError := MsgJSONDecode.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrNoImageReturned):
			wrappedError := MsgNoImageReturned.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		case errors.Is(err, ErrBase64Decode), errors.Is(err, ErrWriteFile):
			wrappedError := MsgSaveFile.Wrap(err)
			ProcessMessage(ctx, b, wrappedError, update.Message.Chat.ID)
			return
		}
	}

}

func main() {
	var inputFile = flag.String("input", "", "Path to input json file")
	flag.Parse()
	inputData, err := os.ReadFile(*inputFile)
	if err != nil {
		panic(err)
	}

	var inputConfig InputConfig
	if err = json.Unmarshal(inputData, &inputConfig); err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	h := Handler{
		ChatGPTClient:  ChatGPTClient{ChatGPTAPIKey: inputConfig.ChatGPTAPIKey, ChatGPTBaseURL: inputConfig.ChatGPTBaseURL},
		TelegramClient: TelegramClient{TelegramFileBaseURL: inputConfig.TelegramFileBaseURL},
	}
	opts := []bot.Option{
		bot.WithDefaultHandler(h.handler),
	}

	b, err := bot.New(inputConfig.TelegramAPIKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func photoProcessingChatGPT(apiKey string, imgPath string, baseURL string) error {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProcessFile, err)
	}
	defer imgFile.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("model", "gpt-image-1")
	_ = writer.WriteField("prompt", "here is a picture of a dog, you need to cut the dog out of the picture and make a png file")
	_ = writer.WriteField("size", "1024x1024")
	_ = writer.WriteField("background", "transparent")

	imgPart, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {fmt.Sprintf(`form-data; name="image"; filename="%s"`, filepath.Base(imgPath))},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMultipartCreatePart, err)
	}
	if _, err = io.Copy(imgPart, imgFile); err != nil {
		return fmt.Errorf("%w: %v", ErrMultipartWriteFile, err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrCloseFile, err)
	}

	req, err := http.NewRequest("POST", baseURL, &buf)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreateRequestChatGPT, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendRequestChatGPT, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status: %v,%s", ErrBadStatusCodeChatGPT, resp.Status, string(body))
	}

	var out imagesResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("%w: %v", ErrJSONDecode, err)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return fmt.Errorf("%w", ErrNoImageReturned)
	}

	bytesPNG, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBase64Decode, err)
	}
	if err = os.WriteFile("dog_cutout.png", bytesPNG, 0644); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteFile, err)
	}
	return nil
}

type imagesResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}
