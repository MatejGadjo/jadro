package clustering

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	EmbeddingDimension       = 384
	DefaultMaxSequenceLength = 256
	DefaultBatchSize         = 8
)

// onnxResources groups the ONNX session and its pre-allocated tensors so they
// can be created and destroyed together.
type onnxResources struct {
	session             *ort.AdvancedSession
	inputIDsTensor      *ort.Tensor[int64]
	attentionMaskTensor *ort.Tensor[int64]
	tokenTypeIDsTensor  *ort.Tensor[int64]
	outputTensor        *ort.Tensor[float32]
}

func (r *onnxResources) destroy() {
	if r.session != nil {
		r.session.Destroy()
	}
	if r.inputIDsTensor != nil {
		r.inputIDsTensor.Destroy()
	}
	if r.attentionMaskTensor != nil {
		r.attentionMaskTensor.Destroy()
	}
	if r.tokenTypeIDsTensor != nil {
		r.tokenTypeIDsTensor.Destroy()
	}
	if r.outputTensor != nil {
		r.outputTensor.Destroy()
	}
}

// EmbeddingGenerator generates semantic embeddings using ONNX Runtime.
type EmbeddingGenerator struct {
	resources   *onnxResources
	tokenizer   *sentenceTokenizer
	initialized bool
	mu          sync.Mutex
	log         *logger.Logger
	batchSize   int
}

// NewEmbeddingGenerator creates an embedding generator backed by an ONNX model.
// If ONNX is unconfigured or the model files are missing, it returns an
// uninitialized generator that silently produces no embeddings.
func NewEmbeddingGenerator(cfg *config.Config, log *logger.Logger) (*EmbeddingGenerator, error) {
	gen := &EmbeddingGenerator{log: log, batchSize: DefaultBatchSize}

	if cfg.ONNX.RuntimeLibPath == "" {
		log.Warnw("ONNX_RUNTIME_LIB_PATH not set, embeddings disabled")
		return gen, nil
	}

	if _, err := os.Stat(cfg.ONNX.ModelPath); os.IsNotExist(err) {
		log.Warnw("ONNX model file not found, embeddings disabled", "path", cfg.ONNX.ModelPath)
		return gen, nil
	}

	tokenizer, err := loadTokenizer(cfg.ONNX.TokenizerPath, log)
	if err != nil {
		log.Warnw("failed to load tokenizer, embeddings disabled", "error", err)
		return gen, nil
	}

	resources, err := initSession(cfg, gen.batchSize)
	if err != nil {
		log.Warnw("failed to initialize ONNX session, embeddings disabled", "error", err)
		return gen, nil
	}

	gen.tokenizer = tokenizer
	gen.resources = resources
	gen.initialized = true

	log.Infow("ONNX embedding generator initialized",
		"model_path", cfg.ONNX.ModelPath,
		"tokenizer_path", cfg.ONNX.TokenizerPath)

	return gen, nil
}

func initSession(cfg *config.Config, batchSize int) (*onnxResources, error) {
	ort.SetSharedLibraryPath(cfg.ONNX.RuntimeLibPath)
	if !ort.IsInitialized() {
		if err := ort.InitializeEnvironment(); err != nil {
			return nil, fmt.Errorf("initialize ONNX Runtime: %w", err)
		}
	}

	inputInfo, outputInfo, err := ort.GetInputOutputInfo(cfg.ONNX.ModelPath)
	if err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}
	if len(inputInfo) < 2 || len(inputInfo) > 3 || len(outputInfo) < 1 {
		return nil, fmt.Errorf("unexpected model input/output count: inputs=%d outputs=%d", len(inputInfo), len(outputInfo))
	}

	r := &onnxResources{}
	shape := ort.NewShape(int64(batchSize), int64(DefaultMaxSequenceLength))

	r.inputIDsTensor, err = ort.NewEmptyTensor[int64](shape)
	if err != nil {
		return nil, fmt.Errorf("create input IDs tensor: %w", err)
	}

	r.attentionMaskTensor, err = ort.NewEmptyTensor[int64](shape)
	if err != nil {
		r.destroy()
		return nil, fmt.Errorf("create attention mask tensor: %w", err)
	}

	inputNames := []string{inputInfo[0].Name, inputInfo[1].Name}
	outputNames := []string{outputInfo[0].Name}

	if len(inputInfo) == 3 {
		r.tokenTypeIDsTensor, err = ort.NewEmptyTensor[int64](shape)
		if err != nil {
			r.destroy()
			return nil, fmt.Errorf("create token type IDs tensor: %w", err)
		}
		inputNames = append(inputNames, inputInfo[2].Name)
	}

	outputShape := ort.NewShape(int64(batchSize), int64(DefaultMaxSequenceLength), int64(EmbeddingDimension))
	r.outputTensor, err = ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		r.destroy()
		return nil, fmt.Errorf("create output tensor: %w", err)
	}

	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		r.destroy()
		return nil, fmt.Errorf("create session options: %w", err)
	}

	inputs := []ort.Value{r.inputIDsTensor, r.attentionMaskTensor}
	if r.tokenTypeIDsTensor != nil {
		inputs = append(inputs, r.tokenTypeIDsTensor)
	}

	r.session, err = ort.NewAdvancedSession(
		cfg.ONNX.ModelPath,
		inputNames,
		outputNames,
		inputs,
		[]ort.Value{r.outputTensor},
		sessionOptions,
	)
	if err != nil {
		r.destroy()
		return nil, fmt.Errorf("create ONNX session: %w", err)
	}

	return r, nil
}

// IsInitialized reports whether the ONNX model loaded successfully.
func (eg *EmbeddingGenerator) IsInitialized() bool {
	return eg.initialized
}

// hasTokenizer reports whether a tokenizer is loaded and ready.
func (eg *EmbeddingGenerator) hasTokenizer() bool {
	return eg != nil && eg.tokenizer != nil
}

// tokenCount returns the number of tokens in text.
func (eg *EmbeddingGenerator) tokenCount(text string) (int, error) {
	if !eg.hasTokenizer() {
		return 0, fmt.Errorf("tokenizer not initialized")
	}
	return eg.tokenizer.tokenCount(text)
}

// truncateTextToTokens truncates text to at most maxTokens tokens.
func (eg *EmbeddingGenerator) truncateTextToTokens(text string, maxTokens int) (string, error) {
	if !eg.hasTokenizer() {
		return text, fmt.Errorf("tokenizer not initialized")
	}
	return eg.tokenizer.truncateTextToTokens(text, maxTokens)
}

// GenerateEmbedding returns a normalized 384-dimension embedding for text.
func (eg *EmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := eg.generateEmbeddingsBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}
	return embeddings[0], nil
}

func (eg *EmbeddingGenerator) generateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if !eg.initialized || len(texts) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	eg.mu.Lock()
	defer eg.mu.Unlock()

	r := eg.resources
	maxLen := DefaultMaxSequenceLength
	batchSize := eg.batchSize
	inputIDsData := r.inputIDsTensor.GetData()
	attentionMaskData := r.attentionMaskTensor.GetData()
	var tokenTypeIDsData []int64
	if r.tokenTypeIDsTensor != nil {
		tokenTypeIDsData = r.tokenTypeIDsTensor.GetData()
	}

	results := make([][]float32, 0, len(texts))

	for start := 0; start < len(texts); start += batchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(start+batchSize, len(texts))
		count := end - start

		for i := 0; i < batchSize; i++ {
			rowOffset := i * maxLen
			rowIDs := inputIDsData[rowOffset : rowOffset+maxLen]
			rowMask := attentionMaskData[rowOffset : rowOffset+maxLen]

			if i < count {
				inputIDs, attentionMask, err := eg.tokenizer.encode(texts[start+i], maxLen)
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

		runStart := time.Now()
		if err := r.session.Run(); err != nil {
			return nil, fmt.Errorf("ONNX inference failed: %w", err)
		}
		if d := time.Since(runStart); d > 2*time.Second {
			eg.log.Warnw("slow ONNX inference", "duration_ms", d.Milliseconds(), "batch", count)
		}

		outputData := r.outputTensor.GetData()
		if expected := batchSize * maxLen * EmbeddingDimension; len(outputData) != expected {
			return nil, fmt.Errorf("unexpected output size: expected %d, got %d", expected, len(outputData))
		}

		for i := 0; i < count; i++ {
			embedding, err := meanPoolAndNormalize(outputData, attentionMaskData, i, maxLen)
			if err != nil {
				return nil, err
			}
			results = append(results, embedding)
		}
	}

	return results, nil
}

func meanPoolAndNormalize(outputData []float32, attentionMask []int64, batchIdx, maxLen int) ([]float32, error) {
	embedding := make([]float32, EmbeddingDimension)
	rowOffset := batchIdx * maxLen
	var tokenCount float32

	for t := 0; t < maxLen; t++ {
		if attentionMask[rowOffset+t] == 0 {
			continue
		}
		tokenCount++
		base := (rowOffset + t) * EmbeddingDimension
		for d := 0; d < EmbeddingDimension; d++ {
			embedding[d] += outputData[base+d]
		}
	}

	if tokenCount == 0 {
		return nil, fmt.Errorf("no valid tokens for pooling")
	}
	for d := range embedding {
		embedding[d] /= tokenCount
	}

	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	if norm = float32(math.Sqrt(float64(norm))); norm > 0 {
		for d := range embedding {
			embedding[d] /= norm
		}
	}

	return embedding, nil
}
