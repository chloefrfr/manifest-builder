package manifest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
)

type Generator struct {
	chunker *Chunker
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) calculateOptimalChunkSize(rootPath string) (int64, error) {
	var sizes []int64
	var totalSize int64

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		sizes = append(sizes, info.Size())
		totalSize += info.Size()
		return nil
	})

	if err != nil {
		return 0, err
	}
	if len(sizes) == 0 {
		return minChunkSize, nil
	}

	sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })
	p90 := sizes[int(float64(len(sizes))*0.9)]
	avg := totalSize / int64(len(sizes))
	chunkSize := int64(math.Sqrt(float64(p90 * avg)))

	chunkSize = max(min(chunkSize, maxChunkSize), minChunkSize)
	chunkSize = (chunkSize / 65536) * 65536

	if totalChunks := totalSize / chunkSize; totalChunks < 100 {
		chunkSize = totalSize / 100
	}

	return max(chunkSize, minChunkSize), nil
}

func (g *Generator) Generate(rootPath string) (*Manifest, error) {
	startTime := time.Now()

	chunkSize, err := g.calculateOptimalChunkSize(rootPath)
	if err != nil {
		return nil, fmt.Errorf("chunk size calculation failed: %w", err)
	}
	g.chunker = NewChunker(chunkSize)

	workerCount := runtime.NumCPU() * 2
	fileQueue := make(chan string, workerCount*10)
	results := make(chan FileResult, workerCount*10)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup
	var processedFiles atomic.Int64
	var totalFiles atomic.Int64

	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalFiles.Add(1)
		}
		return nil
	})

	fmt.Printf("%s Found %s files\n", cyan("•"), yellow(humanize.Comma(totalFiles.Load())))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileQueue {
				chunks, size, err := g.chunker.Calculate(path)
				if err != nil {
					errChan <- &GenerationError{Path: path, Err: err}
					return
				}

				chunkIDs := make([]int, 0)

				file, err := os.Open(path)
				if err != nil {
					errChan <- &GenerationError{Path: path, Err: err}
					return
				}
				defer file.Close()

				g.chunker.GenerateChunks(path)
				for _, chunk := range chunks {
					chunkIDs = append(chunkIDs, chunk.ChunksIds...)
				}

				relPath, _ := filepath.Rel(rootPath, path)
				results <- FileResult{
					Chunks: chunkIDs,
					Path:   filepath.ToSlash(relPath),
					Size:   size,
				}

				curr := processedFiles.Add(1)
				if curr%100 == 0 {
					fmt.Printf("\r%s Processed %s files (%.1f%%) [%s/s]",
						cyan("•"),
						yellow(humanize.Comma(curr)),
						float64(curr)/float64(totalFiles.Load())*100,
						yellow(humanize.Comma(int64(float64(curr)/time.Since(startTime).Seconds()))))
				}
			}
		}()
	}

	go func() {
		filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			fileQueue <- path
			return nil
		})
		close(fileQueue)
	}()

	var manifest Manifest
	var totalSize int64
	done := make(chan struct{})

	go func() {
		for res := range results {
			manifest.Chunks = append(manifest.Chunks, Chunk{
				ChunksIds: res.Chunks,
				File:      strings.ReplaceAll(res.Path, "/", "\\"),
				FileSize:  res.Size,
			})
			totalSize += res.Size
		}
		close(done)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-done:
		manifest.Name = filepath.Base(rootPath)
		manifest.Size = totalSize
		return &manifest, nil
	}
}

func Write(m *Manifest, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(m)
}
