package slsa

import (
	"io"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
)

type SlsaParser struct{}

func (p *SlsaParser) Parse(r io.Reader) (parser.Mapped, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return parser.Mapped{}, err
	}
	print(b)

	n := parser.Mapped{Source: "SLSA", NormalizedAt: time.Now().Unix()}

	return n, nil
}
