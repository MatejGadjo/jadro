package clustering

import (
	"context"
	"testing"

	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
)

func TestNewEmbeddingGenerator(t *testing.T) {
	log, _ := logger.NewLogger()
	cfg := &config.Config{
		ONNXRuntimeLibPath: "./models/onnxruntime-linux-x64-1.23.2/lib/libonnxruntime.so",
		ONNXModelPath:      "./models/paraphrase-multilingual-MiniLM-L12-v2/model.onnx",
		ONNXTokenizerPath:  "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json",
	}
	gen, err := NewEmbeddingGenerator(cfg, log)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gen == nil {
		t.Error("expected non-nil EmbeddingGenerator")
	}
}

func TestGenerateEmbedding(t *testing.T) {
	log, _ := logger.NewLogger()
	cfg := &config.Config{
		ONNXRuntimeLibPath: "", // Empty to disable embeddings
		ONNXModelPath:      "./models/paraphrase-multilingual-MiniLM-L12-v2/model.onnx",
		ONNXTokenizerPath:  "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json",
	}
	gen, _ := NewEmbeddingGenerator(cfg, log)

	ctx := context.Background()
	embedding, err := gen.GenerateEmbedding(ctx, "новина за политика")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(embedding) != 0 {
		t.Errorf("expected no embedding when ONNX is disabled, got len %d", len(embedding))
	}
}

func TestGenerateEmbedding_EmptyText(t *testing.T) {
	log, _ := logger.NewLogger()
	cfg := &config.Config{
		ONNXRuntimeLibPath: "", // Empty to disable embeddings
		ONNXModelPath:      "./models/paraphrase-multilingual-MiniLM-L12-v2/model.onnx",
		ONNXTokenizerPath:  "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json",
	}
	gen, _ := NewEmbeddingGenerator(cfg, log)

	ctx := context.Background()
	embedding, err := gen.GenerateEmbedding(ctx, "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(embedding) != 0 {
		t.Errorf("expected no embedding when ONNX is disabled, got len %d", len(embedding))
	}
}
