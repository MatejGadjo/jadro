package clustering

import (
	"fmt"
	"os"

	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

const (
	DefaultMaxLength = 512
)

type sentenceTokenizer struct {
	tokenizer *tokenizer.Tokenizer
	maxLength int
	log       *logger.Logger
}

func loadTokenizer(tokenizerPath string, log *logger.Logger) (*sentenceTokenizer, error) {
	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokenizer file not found: %s", tokenizerPath)
	}

	tk, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer from %s: %w", tokenizerPath, err)
	}

	return &sentenceTokenizer{
		tokenizer: tk,
		maxLength: DefaultMaxLength,
		log:       log,
	}, nil
}

func (st *sentenceTokenizer) encode(text string, maxLength int) ([]int64, []int64, error) {
	if maxLength <= 0 {
		maxLength = st.maxLength
	}

	encoding, err := st.tokenizer.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode text: %w", err)
	}

	ids := encoding.Ids
	if len(ids) > maxLength {
		ids = ids[:maxLength]
	}

	inputIDs := make([]int64, maxLength)
	attentionMask := make([]int64, maxLength)

	for i := 0; i < maxLength; i++ {
		if i < len(ids) {
			inputIDs[i] = int64(ids[i])
			attentionMask[i] = 1
		} else {
			inputIDs[i] = 0
			attentionMask[i] = 0
		}
	}

	return inputIDs, attentionMask, nil
}

func (st *sentenceTokenizer) tokenCount(text string) (int, error) {
	encoding, err := st.tokenizer.EncodeSingle(text, false)
	if err != nil {
		return 0, fmt.Errorf("failed to encode text: %w", err)
	}
	return len(encoding.Ids), nil
}

func (st *sentenceTokenizer) truncateTextToTokens(text string, maxTokens int) (string, error) {
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

	return st.tokenizer.Decode(encoding.Ids[:maxTokens], true), nil
}

