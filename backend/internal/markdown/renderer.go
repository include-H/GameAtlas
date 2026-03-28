package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type Renderer struct {
	engine goldmark.Markdown
}

func NewRenderer() *Renderer {
	return &Renderer{
		engine: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				extension.Table,
				extension.Strikethrough,
				extension.TaskList,
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
			),
		),
	}
}

func (r *Renderer) Render(source string) (string, error) {
	var output bytes.Buffer
	if err := r.engine.Convert([]byte(source), &output); err != nil {
		return "", err
	}
	return output.String(), nil
}
