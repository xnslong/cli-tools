package main

import "sync"

type Pool struct {
	data sync.Map
	ch   chan int

	fresh int
}

func NewPool(capacity int) *Pool {
	return &Pool{
		data: sync.Map{},
		ch:   make(chan int, capacity),
	}
}

// GetAll gets all values put in with the Put method
func (p *Pool) GetAll() []string {
	var arr []string
	p.data.Range(func(key, _ any) bool {
		keyStr := key.(string)
		arr = append(arr, keyStr)
		return true
	})
	return arr
}

// Put puts a value. if the value has never been put before, then it's fresh,
// this method will return the fresh rate of the latest few Put operations
func (p *Pool) Put(val string) float64 {
	_, loaded := p.data.LoadOrStore(val, struct{}{})
	fresh := !loaded

	c := freshCount(fresh)
	if cap(p.ch) == 0 {
		return float64(c)
	}

	if len(p.ch) == cap(p.ch) {
		outFresh := <-p.ch
		p.fresh -= outFresh
	}

	p.ch <- c
	p.fresh += c

	r, _ := p.FreshRate()
	return r
}

func freshCount(fresh bool) int {
	if fresh {
		return 1
	} else {
		return 0
	}
}

func (p *Pool) FreshRate() (float64, int) {
	if len(p.ch) > 0 {
		return float64(p.fresh) / float64(len(p.ch)), len(p.ch)
	}

	return 1, len(p.ch)
}
