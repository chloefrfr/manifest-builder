package manifest

import (
	"os"
	"sync/atomic"
)

const (
	minChunkSize   = 512 * 1024
	maxChunkSize   = 10 * 1024 * 1024
	fileBufferSize = 4 * 1024 * 1024
)

type Chunker struct {
	chunkSize int64
	nextChunk atomic.Int64
}

func NewChunker(chunkSize int64, startChunkID int64) *Chunker {
	c := &Chunker{chunkSize: chunkSize}
	c.nextChunk.Store(startChunkID)
	return c
}

func (c *Chunker) Calculate(path string) ([]*Chunk, int64, error) {
	stats, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}

	fileSize := stats.Size()
	if fileSize == 0 {
		return []*Chunk{}, 0, nil
	}

	numChunks := (fileSize + c.chunkSize - 1) / c.chunkSize
	chunks := make([]*Chunk, 0, numChunks)
	chunkIDs := make([]int, 0)

	for offset := int64(0); offset < fileSize; offset += c.chunkSize {
		size := c.chunkSize
		if offset+size > fileSize {
			size = fileSize - offset
		}

		chunkID := int(c.nextChunk.Add(1) - 1)
		chunkIDs = append(chunkIDs, chunkID)
	}

	chunks = append(chunks, &Chunk{
		ChunksIds: chunkIDs,
		File:      path,
		FileSize:  fileSize,
	})

	return chunks, fileSize, nil
}
