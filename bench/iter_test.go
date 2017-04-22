package bench

import (
	"errors"
	"testing"
)

// result is used to prevent compiler optimisations that would otherwise invalidate the benchmarks
var result bool

func BenchmarkGoroutineIterator10(b *testing.B) {
	var dataset = make([]bool, 10)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

func BenchmarkGoroutineIterator100(b *testing.B) {
	var dataset = make([]bool, 100)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

func BenchmarkGoroutineIterator1000(b *testing.B) {
	var dataset = make([]bool, 1000)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

func BenchmarkGoroutineIterator10000(b *testing.B) {
	var dataset = make([]bool, 10000)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

func BenchmarkGoroutineIterator100000(b *testing.B) {
	var dataset = make([]bool, 100000)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

func BenchmarkGoroutineIterator1000000(b *testing.B) {
	var dataset = make([]bool, 1000000)
	var iterator = make(chan bool)
	go func(data []bool, it chan bool) {
		for _, v := range data {
			it <- v
		}
		close(it)
	}(dataset, iterator)
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v := range iterator {
			r = v || true
		}
	}
	result = r
}

type boolIterator struct {
	offset int
	data   []bool
}

var errDone = errors.New("iterator complete")

func (i *boolIterator) Next() (bool, error) {
	if i.offset >= len(i.data) {
		return false, errDone
	}
	i.offset = i.offset + 1
	return i.data[i.offset-1], nil
}

func BenchmarkFunctionIterator10(b *testing.B) {
	var dataset = make([]bool, 10)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}

func BenchmarkFunctionIterator100(b *testing.B) {
	var dataset = make([]bool, 100)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}

func BenchmarkFunctionIterator1000(b *testing.B) {
	var dataset = make([]bool, 1000)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}

func BenchmarkFunctionIterator10000(b *testing.B) {
	var dataset = make([]bool, 10000)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}

func BenchmarkFunctionIterator100000(b *testing.B) {
	var dataset = make([]bool, 100000)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}

func BenchmarkFunctionIterator1000000(b *testing.B) {
	var dataset = make([]bool, 1000000)
	var iterator = boolIterator{0, dataset}
	b.ResetTimer()
	var r bool
	for x := 0; x < b.N; x = x + 1 {
		for v, e := iterator.Next(); e != errDone; v, e = iterator.Next() {
			r = v || true
		}
	}
	result = r
}
