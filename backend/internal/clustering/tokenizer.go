package clustering

import (
	"fmt"
	"os"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

const (
	// DefaultMaxLength is the default maximum sequence length for tokenization
	// Based on config.json: max_position_embeddings = 512
	DefaultMaxLength = 512
)

// SentenceTokenizer wraps the sugarme tokenizer for sentence transformer models
type SentenceTokenizer struct {
	tokenizer *tokenizer.Tokenizer
	maxLength int
	log       *logger.Logger
}

// LoadTokenizer loads a tokenizer from a tokenizer.json file
func LoadTokenizer(tokenizerPath string, log *logger.Logger) (*SentenceTokenizer, error) {
	// Check if file exists
	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokenizer file not found: %s", tokenizerPath)
	}

	// Load tokenizer from file
	tk, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer from %s: %w", tokenizerPath, err)
	}

	return &SentenceTokenizer{
		tokenizer: tk,
		maxLength: DefaultMaxLength,
		log:       log,
	}, nil
}

// Encode tokenizes text and returns token IDs and attention mask
// Returns: (input_ids, attention_mask, error)
func (st *SentenceTokenizer) Encode(text string, maxLength int) ([]int64, []int64, error) {
	if maxLength <= 0 {
		maxLength = st.maxLength
	}

	// Encode the text (add special tokens like [CLS] and [SEP])
	encoding, err := st.tokenizer.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode text: %w", err)
	}

	// Get token IDs
	ids := encoding.Ids

	// Truncate if necessary
	if len(ids) > maxLength {
		ids = ids[:maxLength]
	}

	// Convert to int64
	inputIDs := make([]int64, maxLength)
	attentionMask := make([]int64, maxLength)

	// Copy token IDs and set attention mask
	for i := 0; i < maxLength; i++ {
		if i < len(ids) {
			inputIDs[i] = int64(ids[i])
			attentionMask[i] = 1 // Real token
		} else {
			inputIDs[i] = 0      // Padding token ID (from config.json: pad_token_id = 0)
			attentionMask[i] = 0 // Padding
		}
	}

	return inputIDs, attentionMask, nil
}

// TokenCount returns the number of tokens for the given text without special tokens.
func (st *SentenceTokenizer) TokenCount(text string) (int, error) {
	encoding, err := st.tokenizer.EncodeSingle(text, false)
	if err != nil {
		return 0, fmt.Errorf("failed to encode text: %w", err)
	}

	return len(encoding.Ids), nil
}

// TruncateTextToTokens truncates text to the first maxTokens (excluding special tokens).
func (st *SentenceTokenizer) TruncateTextToTokens(text string, maxTokens int) (string, error) {
	if maxTokens <= 0 || text == "" {
		return "", nil
	}

	encoding, err := st.tokenizer.EncodeSingle(text, false)
	if err != nil {
		return "", fmt.Errorf("failed to encode text: %w", err)
	}

	if len(encoding.Ids) <= maxTokens {
		return text, nil
	}

	truncated := encoding.Ids[:maxTokens]
	return st.tokenizer.Decode(truncated, true), nil
}

// Tokenize is a convenience method that returns just the token IDs
func (st *SentenceTokenizer) Tokenize(text string) ([]int64, error) {
	ids, _, err := st.Encode(text, st.maxLength)
	return ids, err
}
