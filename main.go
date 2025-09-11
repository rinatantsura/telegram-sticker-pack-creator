package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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

const telegramFileBaseURL = "https://api.telegram.org/file/bot%s/%s"
const baseNameOfInputPhoto = "photo_%d.jpg"

var nameToInputPhoto string

type InputConfig struct {
	TelegramAPIKey string `json:"telegram_api_key"`
	TelegramChatID string `json:"telegram_chat_id"`
	ChatGPTAPIKey  string `json:"chat_gpt_api_key"`
}

func main() {
	var inputFile = flag.String("input", "", "Path to input json file")
	flag.Parse()
	inputData, err := os.ReadFile(*inputFile)
	if err != nil {
		panic("Error reading input file")
	}

	var inputConfig InputConfig
	if err := json.Unmarshal(inputData, &inputConfig); err != nil {
		panic("Error parsing input json")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(inputConfig.TelegramAPIKey, opts...)
	if err != nil {
		panic("Error creating bot")
	}

	b.Start(ctx)

	fmt.Println("Input photo:", nameToInputPhoto)

	photoProcessingChatGPT(inputConfig.ChatGPTAPIKey, nameToInputPhoto)
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
		return
	}

	url := fmt.Sprintf(telegramFileBaseURL, b.Token(), file.FilePath)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading file")
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("Error closing response body")
			return
		}
	}()
	if resp.StatusCode != 200 {
		fmt.Println("Error downloading file")
		return
	}

	nameToInputPhoto = fmt.Sprintf(baseNameOfInputPhoto, update.Message.Date)
	out, err := os.Create(nameToInputPhoto)
	if err != nil {
		fmt.Println("Error creating file", err)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			fmt.Println("Error closing output file")
			return
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error copying file", err)
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "I got and saved your photo.",
	})
	if err != nil {
		fmt.Println("Error sending message")
		return
	}

	fmt.Println("Saved photo", nameToInputPhoto)
}

func photoProcessingChatGPT(apiKey string, imgPath string) {

	imgFile, err := os.Open(imgPath)
	if err != nil {
		panic(err)
	}
	defer imgFile.Close()

	fmt.Println("Opened:", imgPath)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Поля формы
	_ = writer.WriteField("model", "gpt-image-1")
	_ = writer.WriteField("prompt", "here is a picture of a dog, you need to cut the dog out of the picture and make a png file")
	_ = writer.WriteField("size", "1024x1024")
	_ = writer.WriteField("background", "transparent") // прозрачный фон

	// Файл изображения
	imgPart, err := writer.CreateFormFile("image", filepath.Base(imgPath))
	if err != nil {
		panic(err)
	}
	if _, err = io.Copy(imgPart, imgFile); err != nil {
		panic(err)
	}

	if err := writer.Close(); err != nil {
		panic(err)
	}

	// 2) HTTP-запрос к /v1/images/edits
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/images/edits", &buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("API error: %s\n%s", resp.Status, string(body)))
	}

	// 3) Парсим ответ и сохраняем PNG
	var out imagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		panic(err)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		panic("no image returned")
	}

	bytesPNG, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("dog_cutout1.png", bytesPNG, 0644); err != nil {
		panic(err)
	}

	fmt.Println("Saved:", "dog_cutout.png")
}

type imagesResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}
