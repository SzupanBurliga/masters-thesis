package security

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var ErrTooComplex = errors.New("payload too complex")

// CheckJSON przechodzi po tokenach JSON bez budowania struktury i odrzuca zbyt głębokie
// zagnieżdżenie oraz zbyt długie tablice.
func CheckJSON(data []byte, maxDepth, maxArray int) error {
	dec := json.NewDecoder(bytesReader(data))
	var depth int
	var counts []int // liczniki elementów w otwartych tablicach (-1 dla obiektów)
	for {
		t, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if d, ok := t.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
				if depth > maxDepth {
					return ErrTooComplex
				}
				if len(counts) > 0 && counts[len(counts)-1] >= 0 {
					counts[len(counts)-1]++
					if counts[len(counts)-1] > maxArray {
						return ErrTooComplex
					}
				}
				if d == '[' {
					counts = append(counts, 0)
				} else {
					counts = append(counts, -1)
				}
			default:
				depth--
				counts = counts[:len(counts)-1]
			}
			continue
		}
		if len(counts) > 0 && counts[len(counts)-1] >= 0 {
			counts[len(counts)-1]++
			if counts[len(counts)-1] > maxArray {
				return ErrTooComplex
			}
		}
	}
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
