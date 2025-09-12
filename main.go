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
const chatGPTBaseURL = "https://api.openai.com/v1/images/edits"

var InputPhotoName string

type InputConfig struct {
	TelegramAPIKey string `json:"telegram_api_key"`
	TelegramChatID string `json:"telegram_chat_id"`
	ChatGPTAPIKey  string `json:"chat_gpt_api_key"`
}

type a struct {
	ChatGPTAPIKey string `json:"chat_gpt_api_key"`
}

func (a a) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || len(update.Message.Photo) == 0 {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please, send me a photo",
		})
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
		return
	}
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
	})
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return
	}

	url := fmt.Sprintf(telegramFileBaseURL, b.Token(), file.FilePath)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading file:", err)
		return
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			fmt.Println("Error closing response body:", err)
			return
		}
	}()
	if resp.StatusCode != 200 {
		fmt.Println("Error downloading file")
		return
	}

	InputPhotoName = fmt.Sprintf(baseNameOfInputPhoto, update.Message.Date)
	out, err := os.Create(InputPhotoName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer func() {
		if err = out.Close(); err != nil {
			fmt.Println("Error closing output file:", err)
			return
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error copying file:", err)
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "I got and saved your photo.",
	})
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Println("Saved photo", InputPhotoName)

	photoProcessingChatGPT(a.ChatGPTAPIKey, InputPhotoName)
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

	a := a{
		ChatGPTAPIKey: inputConfig.ChatGPTAPIKey,
	}
	opts := []bot.Option{
		bot.WithDefaultHandler(a.handler),
	}

	b, err := bot.New(inputConfig.TelegramAPIKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func photoProcessingChatGPT(apiKey string, imgPath string) {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		fmt.Println("Error opening image:", err)
		return
	}
	defer imgFile.Close()

	fmt.Println("Opened:", imgPath)

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
		fmt.Println("Error creating form file:", err)
		return
	}
	if _, err = io.Copy(imgPart, imgFile); err != nil {
		fmt.Println("Error copying image:", err)
		return
	}

	if err = writer.Close(); err != nil {
		fmt.Println("Error closing writer:", err)
		return
	}

	req, err := http.NewRequest("POST", chatGPTBaseURL, &buf)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("API error: ", resp.Status, string(body))
		return
	}

	var out imagesResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		fmt.Println("no image returned")
		return
	}

	bytesPNG, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		fmt.Println("Error decoding base64:", err)
		return
	}
	if err = os.WriteFile("dog_cutout.png", bytesPNG, 0644); err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Saved:", "dog_cutout.png")
}

type imagesResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}
