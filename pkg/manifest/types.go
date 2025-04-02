package manifest

import "fmt"

type Manifest struct {
	Name   string  `json:"Name"`
	Chunks []Chunk `json:"Chunks"`
	Size   int64   `json:"Size"`
}

type Chunk struct {
	ChunksIds []int  `json:"ChunksIds"`
	File      string `json:"File"`
	FileSize  int64  `json:"FileSize"`
}

type FileResult struct {
	Chunks []int
	Path   string
	Size   int64
}

type GenerationError struct {
	Path string
	Err  error
}

func (e *GenerationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}