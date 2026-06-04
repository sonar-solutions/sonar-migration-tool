// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package gui

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// mdRenderer is a goldmark instance with table and strikethrough extensions.
var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.Table),
)

// RenderMarkdown converts markdown text to HTML using goldmark.
func RenderMarkdown(md string) (string, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
