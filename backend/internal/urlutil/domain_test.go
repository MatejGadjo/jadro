package urlutil

import "testing"

func TestDomain(t *testing.T) {
	cases := map[string]string{
		"":                                "",
		"   ":                             "",
		"https://www.Example.mk/feed/":    "example.mk",
		"http://example.mk:8080/a?b=c":    "example.mk",
		"example.mk":                      "example.mk",
		"www.example.mk/sport/news":       "example.mk",
		"example.mk:443":                  "example.mk",
		"https://sub.example.mk":          "sub.example.mk",
		"  HTTPS://WWW.EXAMPLE.MK/FEED  ": "example.mk",
	}
	for in, want := range cases {
		if got := Domain(in); got != want {
			t.Errorf("Domain(%q) = %q, want %q", in, got, want)
		}
	}
}
