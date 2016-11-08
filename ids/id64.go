package ids

import (
	"fmt"
	"math/rand"
	"time"
)

// ID64 implements ID interface
type ID64 int64

var rnd *rand.Rand

// NewID64 generates a random 64 bits
func NewID64() ID64 {
	if rnd == nil {
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return ID64(rnd.Int63())
}

func (id ID64) GetID() (interface{}, error) {
	return id, nil
}

type ID64Factory struct{}

func NewID64Factory() *ID64Factory {
	return &ID64Factory{}
}

func (f *ID64Factory) NewID() interface{} {
	return NewID64()
}

func (f *ID64Factory) CreateID() interface{} {
	return ID64(0)
}

func (f *ID64Factory) ParseID(v interface{}) (interface{}, error) {
	switch obj := v.(type) {
	case int64:
		return ID64(obj), nil
	case ID64:
		return obj, nil
	}
	return nil, fmt.Errorf("Impossible to parse in ID64 %v", v)
}
