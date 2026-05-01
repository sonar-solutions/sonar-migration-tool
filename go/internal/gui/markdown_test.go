package gui

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	md := "# Hello\n\nThis is **bold** text."
	html, err := RenderMarkdown(md)
	if err != nil {
		t.Fatalf("RenderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Errorf("missing h1 tag in: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("missing strong tag in: %s", html)
	}
}

func TestRenderMarkdownTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |"
	html, err := RenderMarkdown(md)
	if err != nil {
		t.Fatalf("RenderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "<table>") {
		t.Errorf("missing table tag in: %s", html)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	html, err := RenderMarkdown("")
	if err != nil {
		t.Fatalf("RenderMarkdown error: %v", err)
	}
	if html != "" {
		t.Errorf("expected empty string, got: %q", html)
	}
}
