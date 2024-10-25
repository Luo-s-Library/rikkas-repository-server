package books

type Book struct {
	Title    string
	Sections []Section
	Images   []string
}

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

type Section struct {
	IsImage    bool
	ImageUrl   string
	Text       string
	Tokens     []Token
	HasWavFile bool
	WavFileUrl string
}

type Token struct {
	Text     string
	Furigana string
	Features []string
}

type BookShelf struct {
	Books []BookLink
}

type BookLink struct {
	Title      string
	CoverImage string
	SoundFiles string
}
