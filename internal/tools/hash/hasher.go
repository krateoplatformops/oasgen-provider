package hasher

import (
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
)

type ObjectHash struct {
	hash.Hash64
}

// the hash is cumulative, so you can call Hash() multiple times
// with different values and the hash will be updated
func (h *ObjectHash) SumHash(a ...any) error {
	for _, v := range a {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := h.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func (h *ObjectHash) Reset() {
	h.Hash64.Reset()
}
func (h *ObjectHash) GetHash() string {
	return fmt.Sprintf("%x", h.Hash64.Sum64())
}

func NewFNVObjectHash() ObjectHash {
	return ObjectHash{fnv.New64()}
}
