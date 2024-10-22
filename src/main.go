package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rikkas-repository/books"
	"rikkas-repository/storage"
	"strings"
	"unicode"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go/aws"
)

func DownloadMP3IfNotExists(text string, filePath string) error {
	if Exists(filePath) {
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

func IsWhitespaceOnly(text string) bool {
	for _, r := range text {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func ProcessHtml(n *html.Node, book *books.Book) string {
	if n.Data == "head" {
		return ""
	}
	if n.Data == "body" {
		for _, d := range n.Attr {
			if d.Key == "class" {
				if d.Val == "p-caution" || d.Val == "p-colophon" {
					return ""
				}
			}
		}
	}
	if n.Type == html.ElementNode && n.Data == "img" {
		for _, d := range n.Attr {
			if d.Key == "src" {
				book.Sections = append(book.Sections, *books.NewSection())
				book.Sections[len(book.Sections)-1].ImageUrl = "images/" + filepath.Base(d.Val)
				book.Sections[len(book.Sections)-1].IsImage = true
				if !contains(book.Images, filepath.Base(d.Val)) {
					book.Images = append(book.Images, filepath.Base(d.Val))
				}
				return ""
			}
		}
	}
	if n.Type == html.ElementNode && n.Data == "ruby" {
		var kanjiText string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				kanjiText += c.Data
			} else if c.Type == html.ElementNode && c.Data == "rt" {
				continue
			}
		}
		return kanjiText
	} else if n.Type == html.TextNode {
		text := n.Data

		if IsWhitespaceOnly(text) {
			return ""
		}

		return strings.TrimSpace(text)
	} else {
		text := ""
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			text += ProcessHtml(c, book)
		}
		if n.Type == html.ElementNode && n.Data == "p" {
			content := ""
			inQuote := false
			for _, char := range text {
				content += string(char)
				if char == '「' {
					inQuote = true
				} else if char == '」' {
					inQuote = false
				} else if char == '。' {
					if !inQuote {
						book.Sections = append(book.Sections, *books.NewSection())
						book.Sections[len(book.Sections)-1].Text = content
						content = ""
					}
				}
			}
			if content != "" {
				book.Sections = append(book.Sections, *books.NewSection())
				book.Sections[len(book.Sections)-1].Text = content
			}

		}
	}
	return ""
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CreateDirectoryIfNotExists(path string) {
	if !Exists(path) {
		os.Mkdir(path, 0755)
	}
}

func DownloadEPUB(r *http.Request, tempDir string) (string, error) {
	// Only allow files of certain size?? idk what this line does tbh
	r.ParseMultipartForm(10 << 20)

	// Retrieve the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("error retrieving the file: %w", err)
	}
	defer file.Close()

	// Ensure the uploaded file is an epub
	if filepath.Ext(handler.Filename) != ".epub" {
		return "", fmt.Errorf("uploaded file was not .epub")
	}

	// Create the temporary directory
	CreateDirectoryIfNotExists("./" + tempDir + "/")
	dst, err := os.Create("./" + tempDir + "/" + handler.Filename)
	if err != nil {
		return "", fmt.Errorf("error creating file space: %w", err)
	}
	defer dst.Close()

	// Save the upladed file to the temp directory
	_, err = io.Copy(dst, file)
	if err != nil {
		return "", fmt.Errorf("error saving file: %w", err)
	}
	return handler.Filename, nil
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	tempDir := "downloading"
	mp3Dir := "mp3"
	imagesDir := "images"
	defer CleanUp(tempDir)

	// Download .epub file
	filename, err := DownloadEPUB(r, tempDir)
	if err != nil {
		fmt.Println("Error Getting EPUB")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Open the .epub file for processing
	rc, err := epub.OpenReader("./" + tempDir + "/" + filename)
	if err != nil {
		fmt.Println("Error opening epub file")
		http.Error(w, "Failed to parse epub file", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	book := rc.Rootfiles[0]

	processedBook := books.Book{
		Title:    book.Title,
		Sections: []books.Section{},
	}

	if storage.HasBook(book.Title) {
		http.Error(w, "Book has already been uploaded", http.StatusInternalServerError)
		return
	}

	// Make the Directory for the data to be stored in
	CreateDirectoryIfNotExists("./" + tempDir + "/" + processedBook.Title)

	// Create the image directory
	CreateDirectoryIfNotExists("./" + tempDir + "/" + processedBook.Title + "/" + imagesDir)

	// Create the mp3 directory
	CreateDirectoryIfNotExists("./" + tempDir + "/" + processedBook.Title + "/" + mp3Dir)

	// Iterate through the spine of the book
	for _, section := range book.Spine.Itemrefs {
		// Open the item
		f, err := section.Open()
		if err != nil {
			fmt.Println("Error opening a section")
			http.Error(w, "unable to parse epub file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// Setup the parser
		doc, err := html.Parse(f)
		if err != nil {
			fmt.Println("Error parsing the html")
			http.Error(w, "Error parsing html", http.StatusInternalServerError)
			return
		}

		// Parse the document
		ProcessHtml(doc, &processedBook)
	}

	// Create the tokenizer for processing text
	t := tokenizer.New()

	// Initialize counter for mp3 files
	mp3Files := 1

	progress := 0

	fmt.Println("Beginning to Process Each Segments")
	// Process Each Segment
	for i, sentence := range processedBook.Sections {
		if sentence.IsImage {
			// Ensure the image is included in the new book format
			for _, img := range book.Manifest.Items {
				if filepath.Base(img.HREF) != filepath.Base(sentence.ImageUrl) {
					continue
				}

				f, err := img.Open()
				if err != nil {
					fmt.Println("Error opening image: %w", err)
					http.Error(w, "Error parsing epub", http.StatusInternalServerError)
					return
				}
				defer f.Close()

				err = CopyImageIfNotExists(&f, "./"+tempDir+"/"+processedBook.Title+"/"+imagesDir+"/"+filepath.Base(sentence.ImageUrl))
				if err != nil {
					fmt.Println("Error Copying the image: %w", err)
					http.Error(w, "Error copying images", http.StatusInternalServerError)
					return
				}
			}
		} else {
			// Get the tokens of the text
			tokens := t.Analyze(sentence.Text, tokenizer.Normal)
			for _, token := range tokens {
				if token.Class == tokenizer.DUMMY {
					continue
				}
				processedBook.Sections[i].Tokens = append(processedBook.Sections[i].Tokens, books.Token{
					Text:     token.Surface,
					Features: token.Features(),
				})
			}

			// Download the mp3 of the synthesis from aws
			mp3FileName := fmt.Sprintf("%0*d", 7, mp3Files)
			mp3Files++
			filePath := "./" + tempDir + "/" + processedBook.Title + "/" + mp3Dir + "/" + mp3FileName + ".mp3"
			err = DownloadMP3IfNotExists(sentence.Text, filePath)
			if err != nil {
				fmt.Println("Error Downloading mp3 file from aws")
				http.Error(w, "Error Downloading mp3 file from aws", http.StatusInternalServerError)
				return
			}
			processedBook.Sections[i].WavFileUrl = mp3FileName + ".mp3"
			processedBook.Sections[i].HasWavFile = true
		}
		progress++
		fmt.Printf("\rProgress: %s", fmt.Sprintf("%0*f", 4, float64(progress)/float64(len(processedBook.Sections))*100))

	}

	// Convert Processed Book to JSON Format
	jsonData, err := json.MarshalIndent(processedBook, "", " ")
	if err != nil {
		fmt.Println("Error processing epub")
		http.Error(w, "Error processing epub", http.StatusInternalServerError)
		return
	}

	// Write the json to a file
	err = os.WriteFile("./"+tempDir+"/"+processedBook.Title+"/data.json", jsonData, 0644)
	if err != nil {
		fmt.Println("Error saving file")
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// zip the file
	zipFilePath := "./" + tempDir + "/" + processedBook.Title + ".zip"
	fileSize, err := ZipFolder("./"+tempDir+"./"+processedBook.Title, zipFilePath)
	if err != nil {
		fmt.Println("Error zipping file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Save it to "S3"
	localUrl, err := storage.SaveFileToS3Bucket(zipFilePath)
	if err != nil {
		fmt.Println("Failed to save zip file to S3 bucket")
		http.Error(w, "Failed to save to storage", http.StatusInternalServerError)
		return
	}

	// Add the entry to the database
	storage.AddBook(localUrl, book.Title, string(fileSize))

}

func CleanUp(tempDir string) {
	err := os.RemoveAll("./" + tempDir)
	if err != nil {
		log.Fatal(err)
	}
}

func ZipFolder(source, target string) (int64, error) {
	zipFile, err := os.Create(target)
	if err != nil {
		return 0, err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name, _ = filepath.Rel(filepath.Dir(source), path)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	stat, err := os.Stat(target)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func CopyImageIfNotExists(f *io.ReadCloser, dst string) error {
	if Exists(dst) {
		return nil
	}
	imgData, _, err := image.Decode(*f)
	if err != nil {
		return fmt.Errorf("error Decoding image: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating output image: %w", err)
	}
	defer dstFile.Close()

	err = png.Encode(dstFile, imgData)
	if err != nil {
		return fmt.Errorf("error encoding new image: %w", err)
	}
	return nil
}

func main() {
	// Initialize database
	storage.InitializeRepository()

	http.HandleFunc("/upload", uploadFile)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/index.html")
	})

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/style.css")
	})

	http.HandleFunc("/getbook", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Serving book")
		title := r.URL.Query().Get("title")

		if title == "" {
			fmt.Println("Book not found")
			http.Error(w, "Title parameter is missing", http.StatusBadRequest)
			return
		}

		book := storage.GetBook(title)
		if book == nil {
			http.Error(w, "Title not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, book.LocalUrl)
	})

	http.HandleFunc("/getbooklist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bookshelf, err := storage.GetAllBooks()
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(bookshelf)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
