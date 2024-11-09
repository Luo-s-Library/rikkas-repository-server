package storage

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"rikkas-repository/books"

	_ "modernc.org/sqlite"
)

var COL_TITLE = "title"
var COL_COVER_IMG = "coverImage"
var COL_SOUND_FILE_STATUS = "soundFileStatus"

func Initialize() {
	InitializeRepository()
	CreateDirectoryIfNotExists("./downloading")
	CreateDirectoryIfNotExists("./books")
	CreateDirectoryIfNotExists("./books/temp")
}

/* Local File Storage */
func CreateDirectoryTreeForBook(title string) {
	CreateDirectoryIfNotExists("./books/temp/" + title)
	CreateDirectoryIfNotExists("./books/temp/" + title + "/images")
	CreateDirectoryIfNotExists("./books/temp/" + title + "/mp3")
}

func ClearCacheForBook(title string) {
	os.RemoveAll("./books/temp/" + title)
}

func DeleteFile(filename string) {
	os.Remove(filename)
}

func CopyImageFromEpubToLocalDir(f *io.ReadCloser, dst string) error {
	if Exists(dst) {
		return nil
	}
	imgData, _, err := image.Decode(*f)
	if err != nil {
		return fmt.Errorf("error decoding image: %w", err)
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

func Unzip(src, dst string) error {
	// Open the ZIP file
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through the files in the archive
	for _, f := range r.File {
		// Create the full path for the destination file
		dstFile := filepath.Join(dst, f.Name)

		// Check if the file is a directory
		if f.FileInfo().IsDir() {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(dstFile, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Create the destination file
		if err := os.MkdirAll(filepath.Dir(dstFile), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.Create(dstFile)
		if err != nil {
			return err
		}

		// Open the ZIP file
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// Copy the content to the destination file
		_, err = io.Copy(outFile, rc)
		rc.Close() // Close the reader
		if err != nil {
			outFile.Close()
			return err
		}

		outFile.Close() // Close the destination file
	}

	return nil
}

func ZipFolder(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
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
	if err != nil {
		return err
	}

	return nil
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

func CopyFile(src, dst string) error {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Sync to flush to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func ZipBook(book books.Book) error {
	// Convert Processed Book to JSON Format
	jsonData, err := json.MarshalIndent(book, "", " ")
	if err != nil {
		return err
	}

	// Write the json to a file
	err = os.WriteFile("./books/temp/"+book.Title+"/data.json", jsonData, 0644)
	if err != nil {
		return err
	}

	CreateDirectoryIfNotExists("./books/" + book.Title + "/")

	// zip the file
	zipFilePath := "./books/" + book.Title + "/" + book.Title + ".zip"
	err = ZipFolder("./books/temp/"+book.Title, zipFilePath)
	if err != nil {
		return err
	}

	MoveToDestination(book.Title)

	return nil
}

func MoveToDestination(title string) {
	CreateDirectoryIfNotExists("./books/" + title)

	zipFilePath := "./books/" + title + "/" + title + ".zip"

	os.Rename("./books/temp/"+title+".zip", zipFilePath)
}

/* Access Database */
func ClearRepository() {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`DROP TABLE IF EXISTS Books;`)
	if err != nil {
		log.Fatal(err)
	}
}

func InitializeRepository() {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS Books (
		"%s" TEXT NOT NULL,
		"%s" TEXT NOT NULL,
		"%s" TEXT NOT NULL
	);`, COL_TITLE, COL_COVER_IMG, COL_SOUND_FILE_STATUS)

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initialized Database")
}

func GetAllBooks() (*books.Library, error) {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		return nil, fmt.Errorf("error opening repository: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf(`SELECT "%s", "%s", "%s" FROM Books`, COL_TITLE, COL_COVER_IMG, COL_SOUND_FILE_STATUS)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error quering db: %w", err)
	}
	defer rows.Close()

	var booklist []books.Book

	for rows.Next() {
		var book books.Book

		err = rows.Scan(&book.Title, &book.CoverImage, &book.AudioFileStatus)
		if err != nil {
			return nil, fmt.Errorf("error scanning item: %w", err)
		}

		booklist = append(booklist, book)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &books.Library{Books: booklist}, nil
}

func AddBook(book books.Book) error {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		return fmt.Errorf("error opening repository: %w", err)
	}
	defer db.Close()

	insertSQL := fmt.Sprintf(`INSERT INTO Books (%s, %s, %s) VALUES (?, ?, "NOT_CREATED")`, COL_TITLE, COL_COVER_IMG, COL_SOUND_FILE_STATUS)
	fmt.Printf("Running Query: %s\n", insertSQL)
	_, err = db.Exec(insertSQL, book.Title, book.CoverImage)
	if err != nil {
		return err
	}

	return nil
}

func HasBook(title string) bool {
	bookshelf, err := GetAllBooks()
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range bookshelf.Books {
		if title == book.Title {
			return true
		}
	}
	return false
}

func UpdateBook(book books.Book) error {
	// Open the SQLite database
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Prepare the SQL update statement
	query := fmt.Sprintf(`UPDATE Books SET %s = ? WHERE %s = ?`, COL_SOUND_FILE_STATUS, COL_TITLE)
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// Execute the statement with the provided title and status
	_, err = stmt.Exec(book.AudioFileStatus, book.Title)
	if err != nil {
		return fmt.Errorf("failed to execute statement: %v", err)
	}

	return nil
}

func GetBook(title string) *books.Book {
	bookshelf, err := GetAllBooks()
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range bookshelf.Books {
		if title == book.Title {
			return &book
		}
	}

	return nil
}
