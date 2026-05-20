package clustering

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	// EmbeddingDimension is the standard dimension for sentence transformer embeddings
	EmbeddingDimension = 384
	// DefaultMaxSequenceLength is the default max sequence length for tokenization
	DefaultMaxSequenceLength = 256
	// DefaultBatchSize is the default ONNX inference batch size.
	DefaultBatchSize = 8
)

// EmbeddingGenerator generates semantic embeddings using ONNX Runtime
type EmbeddingGenerator struct {
	session     *ort.AdvancedSession
	tokenizer   *SentenceTokenizer
	initialized bool
	mu          sync.Mutex
	log         *logger.Logger
	batchSize   int

	// Input/output tensors (reused for each inference)
	inputIDsTensor      *ort.Tensor[int64]
	attentionMaskTensor *ort.Tensor[int64]
	tokenTypeIDsTensor  *ort.Tensor[int64]
	outputTensor        *ort.Tensor[float32]
}

// NewEmbeddingGenerator creates a new embedding generator with ONNX Runtime
func NewEmbeddingGenerator(cfg *config.Config, log *logger.Logger) (*EmbeddingGenerator, error) {
	gen := &EmbeddingGenerator{
		log:         log,
		initialized: false,
		batchSize:   DefaultBatchSize,
	}

	// Check if ONNX Runtime library path is set
	if cfg.ONNXRuntimeLibPath == "" {
		log.Warnw("ONNX_RUNTIME_LIB_PATH not set, embeddings disabled")
		return gen, nil // Return uninitialized generator (no embeddings)
	}

	// Check if model file exists
	if _, err := os.Stat(cfg.ONNXModelPath); os.IsNotExist(err) {
		log.Warnw("ONNX model file not found, embeddings disabled", "path", cfg.ONNXModelPath)
		return gen, nil
	}

	// Set the ONNX Runtime library path
	ort.SetSharedLibraryPath(cfg.ONNXRuntimeLibPath)

	// Initialize ONNX Runtime environment
	if !ort.IsInitialized() {
		err := ort.InitializeEnvironment()
		if err != nil {
			log.Warnw("failed to initialize ONNX Runtime, embeddings disabled", "error", err)
			return gen, nil
		}
	}

	// Load tokenizer
	tokenizer, err := LoadTokenizer(cfg.ONNXTokenizerPath, log)
	if err != nil {
		log.Warnw("failed to load tokenizer, embeddings disabled", "error", err)
		return gen, nil
	}
	gen.tokenizer = tokenizer

	// Get model input/output info to determine names
	inputInfo, outputInfo, err := ort.GetInputOutputInfo(cfg.ONNXModelPath)
	if err != nil {
		log.Warnw("failed to get model info, embeddings disabled", "error", err)
		return gen, nil
	}

	if len(inputInfo) < 2 || len(inputInfo) > 3 || len(outputInfo) < 1 {
		log.Warnw("unexpected model input/output count, embeddings disabled",
			"inputs", len(inputInfo), "outputs", len(outputInfo))
		return gen, nil
	}

	inputNames := []string{inputInfo[0].Name, inputInfo[1].Name}
	outputNames := []string{outputInfo[0].Name}

	// Create input tensors (shape: [batch, max_length])
	shape := ort.NewShape(int64(gen.batchSize), int64(DefaultMaxSequenceLength))
	inputIDsTensor, err := ort.NewEmptyTensor[int64](shape)
	if err != nil {
		log.Warnw("failed to create input IDs tensor, embeddings disabled", "error", err)
		return gen, nil
	}

	attentionMaskTensor, err := ort.NewEmptyTensor[int64](shape)
	if err != nil {
		inputIDsTensor.Destroy()
		log.Warnw("failed to create attention mask tensor, embeddings disabled", "error", err)
		return gen, nil
	}

	var tokenTypeIDsTensor *ort.Tensor[int64]
	if len(inputInfo) == 3 {
		tokenTypeIDsTensor, err = ort.NewEmptyTensor[int64](shape)
		if err != nil {
			inputIDsTensor.Destroy()
			attentionMaskTensor.Destroy()
			log.Warnw("failed to create token type IDs tensor, embeddings disabled", "error", err)
			return gen, nil
		}
		inputNames = append(inputNames, inputInfo[2].Name)
	}

	// Create output tensor (shape: [batch, seq_len, 384]) for last_hidden_state
	outputShape := ort.NewShape(int64(gen.batchSize), int64(DefaultMaxSequenceLength), int64(EmbeddingDimension))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		if tokenTypeIDsTensor != nil {
			tokenTypeIDsTensor.Destroy()
		}
		log.Warnw("failed to create output tensor, embeddings disabled", "error", err)
		return gen, nil
	}

	// Create session options
	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		if tokenTypeIDsTensor != nil {
			tokenTypeIDsTensor.Destroy()
		}
		outputTensor.Destroy()
		log.Warnw("failed to create session options, embeddings disabled", "error", err)
		return gen, nil
	}

	// Create ONNX session
	inputs := []ort.Value{inputIDsTensor, attentionMaskTensor}
	if tokenTypeIDsTensor != nil {
		inputs = append(inputs, tokenTypeIDsTensor)
	}
	outputs := []ort.Value{outputTensor}
	session, err := ort.NewAdvancedSession(
		cfg.ONNXModelPath,
		inputNames,
		outputNames,
		inputs,
		outputs,
		sessionOptions,
	)
	if err != nil {
		inputIDsTensor.Destroy()
		attentionMaskTensor.Destroy()
		if tokenTypeIDsTensor != nil {
			tokenTypeIDsTensor.Destroy()
		}
		outputTensor.Destroy()
		sessionOptions.Destroy()
		log.Warnw("failed to create ONNX session, embeddings disabled", "error", err)
		return gen, nil
	}

	gen.session = session
	gen.inputIDsTensor = inputIDsTensor
	gen.attentionMaskTensor = attentionMaskTensor
	gen.tokenTypeIDsTensor = tokenTypeIDsTensor
	gen.outputTensor = outputTensor
	gen.initialized = true

	log.Infow("ONNX embedding generator initialized successfully",
		"model_path", cfg.ONNXModelPath,
		"tokenizer_path", cfg.ONNXTokenizerPath)

	return gen, nil
}

// IsInitialized returns whether the generator is using ONNX Runtime
func (eg *EmbeddingGenerator) IsInitialized() bool {
	return eg.initialized
}

// TokenCount returns the number of tokens for text using the embedding tokenizer.
func (eg *EmbeddingGenerator) TokenCount(text string) (int, error) {
	if eg == nil || eg.tokenizer == nil {
		return 0, fmt.Errorf("tokenizer not initialized")
	}

	return eg.tokenizer.TokenCount(text)
}

// TruncateTextToTokens truncates text to the first maxTokens using the embedding tokenizer.
func (eg *EmbeddingGenerator) TruncateTextToTokens(text string, maxTokens int) (string, error) {
	if eg == nil || eg.tokenizer == nil {
		return text, fmt.Errorf("tokenizer not initialized")
	}

	return eg.tokenizer.TruncateTextToTokens(text, maxTokens)
}

// GenerateEmbedding generates a semantic embedding vector using ONNX model
// Returns a normalized vector (L2 norm = 1) for cosine similarity
func (eg *EmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := eg.GenerateEmbeddingsBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}
	return embeddings[0], nil
}

// GenerateEmbeddingsBatch generates embeddings for a batch of texts using a single ONNX run.
func (eg *EmbeddingGenerator) GenerateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if !eg.initialized {
		// ONNX not available; return no embeddings.
		return nil, nil
	}
	if len(texts) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	eg.mu.Lock()
	defer eg.mu.Unlock()

	maxLen := DefaultMaxSequenceLength
	batchSize := eg.batchSize
	inputIDsData := eg.inputIDsTensor.GetData()
	attentionMaskData := eg.attentionMaskTensor.GetData()
	var tokenTypeIDsData []int64
	if eg.tokenTypeIDsTensor != nil {
		tokenTypeIDsData = eg.tokenTypeIDsTensor.GetData()
	}

	results := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += batchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		count := end - start

		for i := 0; i < batchSize; i++ {
			rowOffset := i * maxLen
			rowIDs := inputIDsData[rowOffset : rowOffset+maxLen]
			rowMask := attentionMaskData[rowOffset : rowOffset+maxLen]

			if i < count {
				inputIDs, attentionMask, err := eg.tokenizer.Encode(texts[start+i], maxLen)
				if err != nil {
					return nil, fmt.Errorf("failed to tokenize text: %w", err)
				}
				copy(rowIDs, inputIDs)
				copy(rowMask, attentionMask)
			} else {
				for j := range rowIDs {
					rowIDs[j] = 0
					rowMask[j] = 0
				}
			}

			if len(tokenTypeIDsData) > 0 {
				rowTokenTypes := tokenTypeIDsData[rowOffset : rowOffset+maxLen]
				for j := range rowTokenTypes {
					rowTokenTypes[j] = 0
				}
			}
		}

		// Run inference for this batch.
		if err := eg.session.Run(); err != nil {
			return nil, fmt.Errorf("failed to run ONNX inference: %w", err)
		}

		outputData := eg.outputTensor.GetData()
		expectedLen := batchSize * maxLen * EmbeddingDimension
		if len(outputData) != expectedLen {
			return nil, fmt.Errorf("unexpected output dimension: expected %d, got %d", expectedLen, len(outputData))
		}

		for i := 0; i < count; i++ {
			rowOffset := i * maxLen
			embedding := make([]float32, EmbeddingDimension)
			var tokenCount float32

			for t := 0; t < maxLen; t++ {
				if attentionMaskData[rowOffset+t] == 0 {
					continue
				}
				tokenCount++
				base := ((i * maxLen) + t) * EmbeddingDimension
				for d := 0; d < EmbeddingDimension; d++ {
					embedding[d] += outputData[base+d]
				}
			}

			if tokenCount == 0 {
				return nil, fmt.Errorf("no valid tokens for pooling")
			}
			for d := 0; d < EmbeddingDimension; d++ {
				embedding[d] /= tokenCount
			}

			var norm float32
			for _, val := range embedding {
				norm += val * val
			}
			norm = float32(math.Sqrt(float64(norm)))
			if norm > 0 {
				for d := range embedding {
					embedding[d] /= norm
				}
			}

			results = append(results, embedding)
		}
	}

	return results, nil
}

// Close releases ONNX Runtime resources
func (eg *EmbeddingGenerator) Close() error {
	eg.mu.Lock()
	defer eg.mu.Unlock()

	if eg.session != nil {
		eg.session.Destroy()
		eg.session = nil
	}

	if eg.inputIDsTensor != nil {
		eg.inputIDsTensor.Destroy()
		eg.inputIDsTensor = nil
	}

	if eg.attentionMaskTensor != nil {
		eg.attentionMaskTensor.Destroy()
		eg.attentionMaskTensor = nil
	}

	if eg.tokenTypeIDsTensor != nil {
		eg.tokenTypeIDsTensor.Destroy()
		eg.tokenTypeIDsTensor = nil
	}

	if eg.outputTensor != nil {
		eg.outputTensor.Destroy()
		eg.outputTensor = nil
	}

	eg.initialized = false
	return nil
}
