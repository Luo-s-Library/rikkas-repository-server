package uploader

import (
	"encoding/json"
	"fmt"
	"io"
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
)

var DownloadDirectory = "./downloading/"

func LinkZipFile(w http.ResponseWriter, r *http.Request) error {
	filename, err := recvFileFromForm(r, ".zip", "./books/temp/")
	if err != nil {
		return err
	}
	//defer storage.DeleteFile(filename)

	title := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	// Unzip to /books/temp/title
	err = storage.Unzip(filename, "./books/temp/")
	if err != nil {
		return err
	}
	//defer storage.ClearCacheForBook(title)

	jsonData, err := os.ReadFile("./books/temp/" + title + "/data.json")
	if err != nil {
		return err
	}

	var book books.Book
	err = json.Unmarshal(jsonData, &book)
	if err != nil {
		return err
	}

	storage.MoveToDestination(title)
	storage.CopyFile("./books/temp/"+title+"/images/"+book.CoverImage, "./books/"+title+"/"+book.CoverImage)

	storage.AddBook(book)

	return nil
}

func UploadBook(w http.ResponseWriter, r *http.Request) error {
	// Get File from Form Request
	filename, err := recvFileFromForm(r, ".epub", DownloadDirectory)
	if err != nil {
		return err
	}
	//defer storage.DeleteFile(filename)

	// Open the .epub Reader
	rc, err := epub.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("ebook format is not supported: %w", err)
	}
	defer rc.Close()

	book := rc.Rootfiles[0]

	// Check for Duplicates
	if storage.HasBook(book.Title) {
		return fmt.Errorf("book has already been uploaded")
	}

	processedBook := books.Book{
		Title:      book.Title,
		CoverImage: "",
		Images:     []string{},
		Chapters: []books.Chapter{
			{
				Sections: []books.Section{},
			},
		},
	}

	// Create dir for unzipping the book
	storage.CreateDirectoryTreeForBook(book.Title)
	defer storage.ClearCacheForBook(book.Title)

	// Process the Book in to our own format
	for i, section := range book.Spine.Itemrefs {
		f, err := section.Open()
		if err != nil {
			return err
		}
		defer f.Close()

		doc, err := html.Parse(f)
		if err != nil {
			return err
		}

		ProcessHtml(doc, &processedBook)
		if i != len(book.Spine.Itemrefs)-1 && len(processedBook.Chapters[len(processedBook.Chapters)-1].Sections) > 0 {
			allContentIsImages := true
			for _, content := range processedBook.Chapters[len(processedBook.Chapters)-1].Sections {
				if !content.IsImage {
					allContentIsImages = false
				}
			}
			if !allContentIsImages {
				processedBook.Chapters = append(processedBook.Chapters, books.Chapter{
					Sections: []books.Section{},
				})
			}
		}
	}

	// Tokenize the Japanese text
	tokenizeBook(&processedBook)

	// Copy the image files from the .epub file to the new dir
	err = copyImages(&processedBook, book.Manifest.Items)
	if err != nil {
		return err
	}

	// Save Cover page
	coverImg, err := SaveCoverImage(book.Title, processedBook.Images)
	if err != nil {
		return err
	}
	processedBook.CoverImage = coverImg

	storage.ZipBook(processedBook)

	// Add the entry to the database
	storage.AddBook(processedBook)

	return nil
}

func recvFileFromForm(r *http.Request, extension string, destination string) (string, error) {
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("there was an issue loading the file: %w", err)
	}
	defer file.Close()

	if filepath.Ext(handler.Filename) != extension {
		return "", fmt.Errorf("uploaded file was not an ebook: %w", err)
	}

	dst, err := os.Create(destination + handler.Filename)
	if err != nil {
		return "", fmt.Errorf("error creating output file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", fmt.Errorf("error saving file: %w", err)
	}

	return destination + handler.Filename, nil
}

func tokenizeBook(book *books.Book) {
	t := tokenizer.New()

	for i, chapter := range book.Chapters {
		for j, sentence := range chapter.Sections {
			if !sentence.IsImage {
				tokens := t.Analyze(sentence.Text, tokenizer.Normal)
				for _, token := range tokens {
					if token.Class == tokenizer.DUMMY {
						continue
					}
					book.Chapters[i].Sections[j].Tokens = append(book.Chapters[i].Sections[j].Tokens, books.Token{
						Text:     token.Surface,
						Features: token.Features(),
					})
				}
			}
		}
	}
}

func SaveCoverImage(title string, images []string) (string, error) {
	if len(images) == 0 {
		return "", fmt.Errorf("no images provided")
	}
	for _, img := range images {
		if strings.Contains(strings.TrimSuffix(img, filepath.Ext(img)), "cover") {
			storage.CreateDirectoryIfNotExists(fmt.Sprintf("./books/%s/", title))
			err := storage.CopyFile("./books/temp/"+title+"/images/"+img, "./books/"+title+"/"+img)
			if err != nil {
				return "", err
			}
			return img, nil
		}
	}
	err := storage.CopyFile("./books/temp/"+title+"/images/"+images[0], "./books/"+title+"/"+images[0])
	if err != nil {
		return "", nil
	}
	return images[0], nil
}

func copyImages(book *books.Book, items []epub.Item) error {
	for _, imageLink := range book.Images {
		for _, img := range items {
			if filepath.Base(img.HREF) != filepath.Base(imageLink) {
				continue
			}

			f, err := img.Open()
			if err != nil {
				return err
			}
			defer f.Close()

			err = storage.CopyImageFromEpubToLocalDir(&f, "./books/temp/"+book.Title+"/images/"+filepath.Base(imageLink))
			if err != nil {
				return err
			}
		}
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
	lastChapter := len(book.Chapters) - 1
	if lastChapter == -1 {
		fmt.Println("Error: number of chapters is 0")
	}
	lastSection := len(book.Chapters[lastChapter].Sections)
	if lastChapter == -1 {
		fmt.Println("Error: number of sections is 0")
	}
	if n.Type == html.ElementNode && (n.Data == "img" || n.Data == "image") {
		for _, d := range n.Attr {
			if d.Key == "src" || d.Key == "href" {
				book.Chapters[lastChapter].Sections = append(book.Chapters[lastChapter].Sections, *books.NewSection())
				book.Chapters[lastChapter].Sections[lastSection].ImageUrl = filepath.Base(d.Val)
				book.Chapters[lastChapter].Sections[lastSection].IsImage = true
				if !contains(book.Images, filepath.Base(d.Val)) {
					book.Images = append(book.Images, filepath.Base(d.Val))
				}
				return ""
			}
		}
	}
	if n.Type == html.ElementNode && n.Data == "ruby" {
		kanjiText := ""
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
						book.Chapters[lastChapter].Sections = append(book.Chapters[lastChapter].Sections, *books.NewSection())
						book.Chapters[lastChapter].Sections[lastSection].Text = content
						content = ""
						lastSection++
					}
				}
			}
			if !IsWhitespaceOnly(content) {
				book.Chapters[lastChapter].Sections = append(book.Chapters[lastChapter].Sections, *books.NewSection())
				book.Chapters[lastChapter].Sections[lastSection].Text = content
			}
		}
	}
	return ""
}
