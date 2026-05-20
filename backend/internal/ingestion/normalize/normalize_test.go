package normalize

import (
	"strings"
	"testing"
)

func TestNormalizeArticle(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		snippet        string
		wantTitle      string
		wantSnippet    string
		wantSnippetLen int
	}{
		{
			name:           "basic normalization",
			title:          "Test Article Title!",
			snippet:        "This is a test snippet.",
			wantTitle:      "test article title!",
			wantSnippet:    "This is a test snippet.",
			wantSnippetLen: 24,
		},
		{
			name:           "title with punctuation",
			title:          "Breaking: News Story (2024)",
			snippet:        "A breaking news story.",
			wantTitle:      "breaking: news story (2024)",
			wantSnippet:    "A breaking news story.",
			wantSnippetLen: 22,
		},
		{
			name:           "title with unicode dashes and boilerplate tag",
			title:          "ВИДЕО: Скопје–Тетово: хаос на патот",
			snippet:        "Ова е тест.",
			wantTitle:      "видео: скопје–тетово: хаос на патот",
			wantSnippet:    "Ова е тест.",
			wantSnippetLen: len("Ова е тест."),
		},
		{
			name:           "title strips trailing known source suffix",
			title:          "ЕУ: Нов пакет санкции - МИА",
			snippet:        "Ова е тест.",
			wantTitle:      "еу: нов пакет санкции - миа",
			wantSnippet:    "Ова е тест.",
			wantSnippetLen: len("Ова е тест."),
		},
		{
			name:           "title decodes html entities and keeps boundaries",
			title:          "Apple &quot;Vision Pro&quot; — цена/датум",
			snippet:        "Ова е тест.",
			wantTitle:      "apple \"vision pro\" — цена/датум",
			wantSnippet:    "Ова е тест.",
			wantSnippetLen: len("Ова е тест."),
		},
		{
			name:           "title removes bracketed video/photo tags but keeps year",
			title:          "Избори (2024) (ФОТО) [ВИДЕО]",
			snippet:        "Ова е тест.",
			wantTitle:      "избори (2024) (фото) [видео]",
			wantSnippet:    "Ова е тест.",
			wantSnippetLen: len("Ова е тест."),
		},
		{
			name:           "snippet with HTML",
			title:          "HTML Test",
			snippet:        "<p>This is a <strong>test</strong> snippet with <a href=\"#\">HTML</a>.</p>",
			wantTitle:      "html test",
			wantSnippet:    "This is a test snippet with HTML.",
			wantSnippetLen: 35,
		},
		{
			name:           "snippet with HTML entities",
			title:          "Entity Test",
			snippet:        "This &amp; that &lt;test&gt;",
			wantTitle:      "entity test",
			wantSnippet:    "This & that <test>",
			wantSnippetLen: len("This & that <test>"),
		},
		{
			name:           "long snippet truncation",
			title:          "Long Snippet Test",
			snippet:        strings.Repeat("word ", 100), // 500 chars
			wantTitle:      "long snippet test",
			wantSnippetLen: 300,
		},
		{
			name:           "title with extra whitespace",
			title:          "  Multiple   Spaces   Here  ",
			snippet:        "Test snippet.",
			wantTitle:      "multiple   spaces   here",
			wantSnippet:    "Test snippet.",
			wantSnippetLen: 13,
		},
		{
			name:           "snippet with newlines and extra spaces",
			title:          "Whitespace Test",
			snippet:        "Line 1\n\nLine 2   with   spaces",
			wantTitle:      "whitespace test",
			wantSnippet:    "Line 1 Line 2 with spaces",
			wantSnippetLen: 25,
		},
		{
			name:           "empty inputs",
			title:          "",
			snippet:        "",
			wantTitle:      "",
			wantSnippet:    "",
			wantSnippetLen: 0,
		},
		{
			name:           "Macedonian characters",
			title:          "Македонска Вест!",
			snippet:        "Ова е тест.",
			wantTitle:      "македонска вест!",
			wantSnippet:    "Ова е тест.",
			wantSnippetLen: len("Ова е тест."),
		},
		{
			name:           "mixed HTML and entities",
			title:          "Mixed Test",
			snippet:        "<p>Text with &amp; entities and <strong>tags</strong>.</p>",
			wantTitle:      "mixed test",
			wantSnippet:    "Text with & entities and tags.",
			wantSnippetLen: 33,
		},
		{
			name:           "snippet exactly 300 chars",
			title:          "Exact Length",
			snippet:        strings.Repeat("a", 300),
			wantTitle:      "exact length",
			wantSnippetLen: 300,
		},
		{
			name:           "snippet over 300 chars with word boundary",
			title:          "Word Boundary",
			snippet:        strings.Repeat("word ", 80), // 400 chars, should truncate at word boundary
			wantTitle:      "word boundary",
			wantSnippetLen: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotSnippet := NormalizeArticle(tt.title, tt.snippet)

			if gotTitle != tt.wantTitle {
				t.Errorf("NormalizeArticle() title = %q, want %q", gotTitle, tt.wantTitle)
			}

			if tt.wantSnippet != "" && gotSnippet != tt.wantSnippet {
				t.Errorf("NormalizeArticle() snippet = %q, want %q", gotSnippet, tt.wantSnippet)
			}

			if len(gotSnippet) > tt.wantSnippetLen {
				t.Errorf("NormalizeArticle() snippet length = %d, want <= %d", len(gotSnippet), tt.wantSnippetLen)
			}

			// Verify snippet doesn't exceed max length
			if len(gotSnippet) > maxSnippetLength {
				t.Errorf("NormalizeArticle() snippet length = %d, exceeds max %d", len(gotSnippet), maxSnippetLength)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple title",
			input: "Test Title",
			want:  "test title",
		},
		{
			name:  "with punctuation",
			input: "Hello, World!",
			want:  "hello, world!",
		},
		{
			name:  "replaces separators with spaces (no merged tokens)",
			input: "COVID-19/Грип: 12:30",
			want:  "covid-19/грип: 12:30",
		},
		{
			name:  "strips leading breaking tag",
			input: "Breaking: News Story (2024)",
			want:  "breaking: news story (2024)",
		},
		{
			name:  "strips trailing known source suffix",
			input: "ЕУ: Нов пакет санкции - МИА",
			want:  "еу: нов пакет санкции - миа",
		},
		{
			name:  "with extra spaces",
			input: "  Multiple   Spaces  ",
			want:  "multiple   spaces",
		},
		{
			name:  "with numbers",
			input: "Article 2024",
			want:  "article 2024",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only punctuation",
			input: "!!!",
			want:  "!!!",
		},
		{
			name:  "Macedonian",
			input: "Македонска Вест!",
			want:  "македонска вест!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTitle(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanSnippet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantLen int
	}{
		{
			name:    "simple text",
			input:   "Simple text snippet.",
			want:    "Simple text snippet.",
			wantLen: len("Simple text snippet."),
		},
		{
			name:    "with HTML tags",
			input:   "<p>Text with <strong>tags</strong>.</p>",
			want:    "Text with tags.",
			wantLen: len("Text with tags."),
		},
		{
			name:    "with HTML entities",
			input:   "Text &amp; more &lt;test&gt;",
			want:    "Text & more <test>",
			wantLen: len("Text & more <test>"),
		},
		{
			name:    "with newlines",
			input:   "Line 1\nLine 2\nLine 3",
			want:    "Line 1 Line 2 Line 3",
			wantLen: len("Line 1 Line 2 Line 3"),
		},
		{
			name:    "with extra spaces",
			input:   "Text   with    spaces",
			want:    "Text with spaces",
			wantLen: len("Text with spaces"),
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantLen: 0,
		},
		{
			name:    "long text truncation",
			input:   strings.Repeat("a", 500),
			wantLen: 300,
		},
		{
			name:    "exactly 300 chars",
			input:   strings.Repeat("a", 300),
			wantLen: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanSnippet(tt.input)

			if tt.want != "" && got != tt.want {
				t.Errorf("cleanSnippet(%q) = %q, want %q", tt.input, got, tt.want)
			}

			if len(got) != tt.wantLen {
				t.Errorf("cleanSnippet(%q) length = %d, want %d", tt.input, len(got), tt.wantLen)
			}

			// Verify it doesn't exceed max length
			if len(got) > maxSnippetLength {
				t.Errorf("cleanSnippet(%q) length = %d, exceeds max %d", tt.input, len(got), maxSnippetLength)
			}
		})
	}
}

func TestNormalizeArticle_EdgeCases(t *testing.T) {
	// Test that normalization is idempotent (applying twice gives same result)
	title := "Test Title!!!"
	snippet := "<p>Test <strong>snippet</strong> &amp; more</p>"

	normTitle1, cleanSnippet1 := NormalizeArticle(title, snippet)
	normTitle2, cleanSnippet2 := NormalizeArticle(normTitle1, cleanSnippet1)

	if normTitle1 != normTitle2 {
		t.Errorf("NormalizeArticle not idempotent for title: %q != %q", normTitle1, normTitle2)
	}

	if cleanSnippet1 != cleanSnippet2 {
		t.Errorf("NormalizeArticle not idempotent for snippet: %q != %q", cleanSnippet1, cleanSnippet2)
	}
}
