package books

func NewSection() *Section {
	return &Section{
		IsImage:    false,
		ImageUrl:   "",
		Text:       "",
		Tokens:     []Token{},
		HasWavFile: false,
		WavFileUrl: "",
	}
}

type Book struct {
	Title           string    `json:"title"`
	CoverImage      string    `json:"coverImage"`
	AudioFileStatus string    `json:"audioFileStatus"`
	HasAudioFiles   bool      `json:"hasAudioFiles`
	Images          []string  `json:"images"`
	Sections        []Section `json:"content"`
}

type Section struct {
	IsImage    bool    `json:"isImage"`
	ImageUrl   string  `json:"imageUrl"`
	Text       string  `json:"text"`
	Tokens     []Token `json:"tokens"`
	HasWavFile bool    `json:"hasAudioFile"`
	WavFileUrl string  `json:"audioFileUrl"`
}

type Token struct {
	Text     string   `json:"text"`
	Features []string `json:"features"`
}

type Library struct {
	Books []Book `json:"books"`
}
