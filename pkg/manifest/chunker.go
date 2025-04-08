package manifest

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
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

func NewChunker(chunkSize int64) *Chunker {
	c := &Chunker{chunkSize: chunkSize}
	c.nextChunk.Store(1) 
	return c
}

func (c *Chunker) ResetChunkCounter() {
	c.nextChunk.Store(1)
}

func (c *Chunker) Calculate(path string) ([]*Chunk, int64, error) {
	fmt.Printf("\nCalculating chunks for: %s\n", path)

	stats, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := stats.Size()
	if fileSize == 0 {
		return []*Chunk{}, 0, nil
	}

	c.ResetChunkCounter()

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
	fmt.Printf("\nGenerating chunks for: %s\n", path)

	c.ResetChunkCounter()

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

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers)
	semaphore := make(chan struct{}, numWorkers)

	for i, chunkID := range chunks[0].ChunksIds {
		wg.Add(1)

		go func(idx int, id int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()

			fmt.Printf("Processing chunk %d of %s\n", id, path)

			offset := int64(idx) * c.chunkSize
			size := c.chunkSize
			if offset+size > fileSize {
				size = fileSize - offset
			}

			sf, err := os.Open(path)
			if err != nil {
				errChan <- fmt.Errorf("failed to open source file for chunk %d: %w", id, err)
				return
			}
			defer sf.Close()

			if _, err := sf.Seek(offset, 0); err != nil {
				errChan <- fmt.Errorf("failed to seek in source file for chunk %d: %w", id, err)
				return
			}

			chunkPath := filepath.Join(chunksDir, fmt.Sprintf("%d.chunk", id))
			chunkFile, err := os.Create(chunkPath)
			if err != nil {
				errChan <- fmt.Errorf("failed to create chunk file %s: %w", chunkPath, err)
				return
			}
			defer chunkFile.Close()

			bufferedWriter := bufio.NewWriterSize(chunkFile, fileBufferSize)
			gzipWriter, err := gzip.NewWriterLevel(bufferedWriter, gzip.BestSpeed)
			if err != nil {
				errChan <- fmt.Errorf("failed to create gzip writer: %w", err)
				return
			}

			if _, err := io.CopyN(gzipWriter, sf, size); err != nil {
				gzipWriter.Close()
				errChan <- fmt.Errorf("failed to copy data for chunk %d: %w", id, err)
				return
			}

			if err := gzipWriter.Close(); err != nil {
				errChan <- fmt.Errorf("failed to close gzip writer for chunk %d: %w", id, err)
				return
			}

			if err := bufferedWriter.Flush(); err != nil {
				errChan <- fmt.Errorf("failed to flush buffer for chunk %d: %w", id, err)
				return
			}

		}(i, chunkID)
	}

	wg.Wait()
	close(errChan)

	var errorFound bool
	for err := range errChan {
		errorFound = true
		fmt.Println("[ERROR]", err)
	}

	if errorFound {
		return nil, 0, fmt.Errorf("errors occurred during chunk generation")
	}

	return chunks, fileSize, nil
}

func ProcessFiles(fileQueue chan string, totalFiles atomic.Int64, processedFiles atomic.Int64, g *Chunker, rootPath string, startTime time.Time) {
	var wg sync.WaitGroup
	errChan := make(chan error, 1000)

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileQueue {
				fmt.Printf("\nProcessing file: %s\n", path)

				g.ResetChunkCounter()

				_, _, err := g.Calculate(path)
				if err != nil {
					errChan <- fmt.Errorf("Error calculating chunks for %s: %w", path, err)
					continue
				}
				g.ResetChunkCounter()

				_, _, err = g.GenerateChunks(path)
				if err != nil {
					errChan <- fmt.Errorf("Error generating chunks for %s: %w", path, err)
					continue
				}

				curr := processedFiles.Add(1)
				if curr%100 == 0 {
					fmt.Printf("\r• Processed %d files (%.1f%%) [%d/s]\n",
						curr,
						float64(curr)/float64(totalFiles.Load())*100,
						int(float64(curr)/time.Since(startTime).Seconds()))
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		fmt.Println("[ERROR]", err)
	}
}
