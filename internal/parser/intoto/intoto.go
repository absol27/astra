package intoto

import (
	"io"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
)

type InTotoParser struct{}

func (p *InTotoParser) Parse(r io.Reader) (parser.Mapped, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return parser.Mapped{}, err
	}
	print(b)
	n := parser.Mapped{Source: "in-toto", NormalizedAt: time.Now().Unix()}

	return n, nil
}
