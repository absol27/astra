package slsa

import (
	"os"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
)

type SlsaParser struct{}

func (p *SlsaParser) Parse(path string) (parser.Mapped, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return parser.Mapped{}, err
	}
	print(b)

	n := parser.Mapped{Source: "SLSA", NormalizedAt: time.Now().Unix()}

	return n, nil
}
