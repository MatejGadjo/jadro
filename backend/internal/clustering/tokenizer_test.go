package clustering

import (
	"testing"

	"github.com/MatejGadjo/jadro/internal/logger"
)

func TestLoadTokenizer(t *testing.T) {
	log, _ := logger.NewLogger()
	tokenizerPath := "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json"

	tk, err := loadTokenizer(tokenizerPath, log)
	if err != nil {
		t.Skipf("skipping test: tokenizer not available: %v", err)
		return
	}

	if tk == nil {
		t.Error("expected non-nil tokenizer")
	}

	if tk.maxLength != DefaultMaxLength {
		t.Errorf("expected maxLength = %d, got %d", DefaultMaxLength, tk.maxLength)
	}
}

func TestEncode(t *testing.T) {
	log, _ := logger.NewLogger()
	tokenizerPath := "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json"

	tk, err := loadTokenizer(tokenizerPath, log)
	if err != nil {
		t.Skipf("skipping test: tokenizer not available: %v", err)
		return
	}

	text := "новина за политика"
	inputIDs, attentionMask, err := tk.encode(text, 128)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(inputIDs) != 128 {
		t.Errorf("expected inputIDs length = 128, got %d", len(inputIDs))
	}

	if len(attentionMask) != 128 {
		t.Errorf("expected attentionMask length = 128, got %d", len(attentionMask))
	}

	// Check that attention mask has 1s for real tokens
	hasRealTokens := false
	for _, mask := range attentionMask {
		if mask == 1 {
			hasRealTokens = true
			break
		}
	}

	if !hasRealTokens {
		t.Error("expected at least one real token in attention mask")
	}
}

func TestEncode_EmptyText(t *testing.T) {
	log, _ := logger.NewLogger()
	tokenizerPath := "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json"

	tk, err := loadTokenizer(tokenizerPath, log)
	if err != nil {
		t.Skipf("skipping test: tokenizer not available: %v", err)
		return
	}

	inputIDs, attentionMask, err := tk.encode("", 128)

	if err != nil {
		t.Fatalf("expected no error for empty text, got %v", err)
	}

	if len(inputIDs) != 128 {
		t.Errorf("expected inputIDs length = 128, got %d", len(inputIDs))
	}

	// Empty text should have all padding (attention mask all 0s or mostly 0s)
	// Actually, tokenizer might add special tokens, so we just check it doesn't crash
	if len(attentionMask) != 128 {
		t.Errorf("expected attentionMask length = 128, got %d", len(attentionMask))
	}
}

func tokenize(st *sentenceTokenizer, text string) ([]int64, error) {
	ids, _, err := st.encode(text, st.maxLength)
	return ids, err
}

func TestSentenceTokenizer_Tokenize(t *testing.T) {
	log, _ := logger.NewLogger()
	tokenizerPath := "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json"

	tk, err := loadTokenizer(tokenizerPath, log)
	if err != nil {
		t.Skipf("skipping test: tokenizer not available: %v", err)
		return
	}

	text := "новина за политика"
	ids, err := tokenize(tk, text)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(ids) == 0 {
		t.Error("expected at least one token ID")
	}
}
