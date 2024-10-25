package polly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"rikkas-repository/books"
	"rikkas-repository/storage"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go/aws"
)

func SynthesizeBook(title string) error {
	// Unzip to /books/temp/title
	err := storage.Unzip("./books/"+title+"/"+title+".zip", "./books/temp/")
	if err != nil {
		return nil
	}

	jsonData, err := os.ReadFile("./books/temp/" + title + "data.json")
	if err != nil {
		return err
	}

	var book books.Book
	err = json.Unmarshal(jsonData, &book)
	if err != nil {
		return err
	}

	numFiles := 0

	for i, section := range book.Sections {
		if section.IsImage {
			continue
		}

		fileName := fmt.Sprintf("%0*d", 7, numFiles)
		numFiles++
		filePath := "./books/temp/" + title + "/mp3Dir/" + fileName + ".mp3"
		err := DownloadMP3IfNotExists(section.Text, filePath)
		if err != nil {
			return err
		}
		book.Sections[i].WavFileUrl = fileName + ".mp3"
		book.Sections[i].HasWavFile = true
	}

	// Re-zip the files and move them back to the books directory
	storage.ZipBook(book)

	return nil
}

func DownloadMP3IfNotExists(text string, filePath string) error {
	if storage.Exists(filePath) {
		return nil
	}
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("could not load credentials: %w", err)
	}

	svc := polly.NewFromConfig(cfg)

	voiceID := types.VoiceIdMizuki
	outputFormat := types.OutputFormatMp3

	output, err := svc.SynthesizeSpeech(context.TODO(), &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		VoiceId:      voiceID,
		OutputFormat: outputFormat,
	})
	if err != nil {
		return fmt.Errorf("could not get synthesized output: %w", err)
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, output.AudioStream)
	if err != nil {
		return fmt.Errorf("could not set data to output file: %w", err)
	}

	return nil
}
