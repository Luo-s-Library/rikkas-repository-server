package books

func NewSection() *Section {
	return &Section{
		IsImage:  false,
		ImageUrl: "",
		Text:     "",
		Tokens:   []Token{},
	}
}

type Library struct {
	Books []Book `json:"books"`
}

type Book struct {
	Title      string    `json:"title"`
	CoverImage string    `json:"coverImage"`
	Images     []string  `json:"-"`
	Chapters   []Chapter `json:"chapters"`
}

type Chapter struct {
	Sections []Section `json:"content"`
}

type Section struct {
	IsImage  bool    `json:"isImage"`
	ImageUrl string  `json:"imageUrl"`
	Text     string  `json:"text"`
	Tokens   []Token `json:"tokens"`
}

type Token struct {
	Text     string   `json:"text"`
	Features []string `json:"features"`
}
