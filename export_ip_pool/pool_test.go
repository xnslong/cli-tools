package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Pool(t *testing.T) {
	const delta = 0.0000001

	var r float64
	p := NewPool(5)
	r = p.Put("a")
	assert.InDelta(t, 1, r, delta)
	r = p.Put("a")
	assert.InDelta(t, 0.5, r, delta)
	r = p.Put("a")
	assert.InDelta(t, 1.0/3, r, delta)
	r = p.Put("a")
	assert.InDelta(t, 1.0/4, r, delta)
	r = p.Put("a")
	assert.InDelta(t, 1.0/5, r, delta)
	r2, total := p.FreshRate()
	assert.InDelta(t, r, r2, delta)
	assert.Equal(t, 5, total)

	//
	r = p.Put("a")
	assert.InDelta(t, 0, r, delta)
}
