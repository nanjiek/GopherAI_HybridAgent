package file

import (
	"context"
	"github.com/nanjiek/GopherMind/common/rag"
	"github.com/nanjiek/GopherMind/config"
	"github.com/nanjiek/GopherMind/utils"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

// UploadRagFile uploads one text file and builds vector index.
func UploadRagFile(username string, file *multipart.FileHeader) (string, error) {
	if err := utils.ValidateFile(file); err != nil {
		log.Printf("File validation failed: %v", err)
		return "", err
	}

	userDir := filepath.Join("uploads", username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Failed to create user directory %s: %v", userDir, err)
		return "", err
	}

	// one user keeps one active RAG file; clear existing files and indexes
	files, err := os.ReadDir(userDir)
	if err == nil {
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			filename := f.Name()
			if err := rag.DeleteIndex(context.Background(), filename); err != nil {
				log.Printf("Failed to delete index for %s: %v", filename, err)
			}
		}
	}

	if err := utils.RemoveAllFilesInDir(userDir); err != nil {
		log.Printf("Failed to clean user directory %s: %v", userDir, err)
		return "", err
	}

	uuid := utils.GenerateUUID()
	ext := filepath.Ext(file.Filename)
	filename := uuid + ext
	filePath := filepath.Join(userDir, filename)

	src, err := file.Open()
	if err != nil {
		log.Printf("Failed to open uploaded file: %v", err)
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create destination file %s: %v", filePath, err)
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		log.Printf("Failed to copy file content: %v", err)
		return "", err
	}

	log.Printf("File uploaded successfully: %s", filePath)

	indexer, err := rag.NewRAGIndexer(filename, config.GetConfig().RagModelConfig.RagEmbeddingModel)
	if err != nil {
		log.Printf("Failed to create RAG indexer: %v", err)
		_ = os.Remove(filePath)
		return "", err
	}

	if err := indexer.IndexFile(context.Background(), filePath); err != nil {
		log.Printf("Failed to index file: %v", err)
		_ = os.Remove(filePath)
		_ = rag.DeleteIndex(context.Background(), filename)
		return "", err
	}

	log.Printf("File indexed successfully: %s", filename)
	return filePath, nil
}
