package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func (c *Chunker) GenerateChunks(path string) ([]*Chunk, int64, error) {
	chunks, fileSize, err := c.Calculate(path)
	if err != nil {
		return nil, 0, err
	}
	chunksDir := "chunks"
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, 0, fmt.Errorf("failed to create chunks directory: %w", err)
	}
	if len(chunks) == 0 {
		return chunks, 0, nil
	}
	sourceFile, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	for i, _ := range chunks[0].ChunksIds {
		offset := int64(i) * c.chunkSize
		size := c.chunkSize
		if offset+size > fileSize {
			size = fileSize - offset
		}

		uuid := generateUUID()
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("%s.chunk", uuid))

		chunkFile, err := os.Create(chunkPath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create chunk file %s: %w", chunkPath, err)
		}

		if _, err := sourceFile.Seek(offset, 0); err != nil {
			chunkFile.Close()
			return nil, 0, fmt.Errorf("failed to seek in source file: %w", err)
		}

		buffer := make([]byte, fileBufferSize)
		bytesRemaining := size
		for bytesRemaining > 0 {
			bufferSize := int64(fileBufferSize)
			if bytesRemaining < bufferSize {
				bufferSize = bytesRemaining
			}
			n, err := sourceFile.Read(buffer[:bufferSize])
			if err != nil && err != io.EOF {
				chunkFile.Close()
				return nil, 0, fmt.Errorf("failed to read from source file: %w", err)
			}
			if n > 0 {
				if _, err := chunkFile.Write(buffer[:n]); err != nil {
					chunkFile.Close()
					return nil, 0, fmt.Errorf("failed to write to chunk file: %w", err)
				}
				bytesRemaining -= int64(n)
			}
			if err == io.EOF {
				break
			}
		}

		if err := chunkFile.Close(); err != nil {
			return nil, 0, fmt.Errorf("failed to close chunk file: %w", err)
		}
	}

	return chunks, fileSize, nil
}
