package chat_gpt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rinatantsura/telegram-sticker-pack-creator/internal/errors"
	"github.com/rs/zerolog/log"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type ClientChatGPT struct {
	APIKey  string `json:"chat_gpt_api_key"`
	BaseURL string `json:"chat_gpt_base_url"`
}

func NewClient(apiKey string, baseUrl string) ClientChatGPT {
	return ClientChatGPT{
		APIKey:  apiKey,
		BaseURL: baseUrl,
	}
}

func (c ClientChatGPT) DeletePhotoBackground(ctx context.Context, imgPath string) (string, error) {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		log.Error().Err(err).Str("file", imgPath).Msg("Failed to open file")
		return "", err
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
		log.Error().Err(err).Msg("Failed to create a new multipart section")
		return "", err
	}
	if _, err = io.Copy(imgPart, imgFile); err != nil {
		log.Error().Err(err).Msg("Failed to copy image file")
		return "", err
	}

	if err = writer.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to сlose writer")
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, &buf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new request")
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send the request")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status: %v,%s", errors.ErrBadStatusCodeChatGPT, resp.Status, string(body))
	}

	var out imagesResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Error().Err(err).Msg("Failed to decode the JSON response from the service")
		return "", err
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		log.Error().Err(err).Msg("The response from the service is empty or does not contain B64JSON data")
		return "", err
	}

	bytesPNG, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		log.Error().Err(err).Msg("The response from the service is empty or does not contain B64JSON data")
		return "", err
	}
	outputPath := "dog_cutout.png"
	if err = os.WriteFile(outputPath, bytesPNG, 0644); err != nil {
		log.Error().Err(err).Msg("Failed to write file")
		return "", err
	}
	return outputPath, nil
}

type imagesResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
}
