package intoto

import (
	"os"
	"time"

	parser "github.com/abuishgair/astra/internal/parser"
)

type InTotoParser struct{}

func (p *InTotoParser) Parse(path string) (parser.Mapped, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return parser.Mapped{}, err
	}
	print(b)
	n := parser.Mapped{Source: "in-toto", NormalizedAt: time.Now().Unix()}

	return n, nil
}
