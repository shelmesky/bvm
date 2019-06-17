package types

type File struct {
	Name     string
	MimeType string
	Body     []byte
}

func NewFile() *File {
	return FileInit("", "", []byte{})
}

func FileInit(name string, mime string, body []byte) *File {
	return &File{
		Name:     name,
		MimeType: mime,
		Body:     body,
	}
}
