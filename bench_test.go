package slabgo_test

import (
	"testing"
	"unsafe"

	"github.com/k-sone/slabgo"
)

type Bar struct {
	num   int
	name  string
	slice []*Bar
	any   interface{}
}

func builtinAllocate(n int) {
	z := make([]*Bar, n)
	for i := 0; i < n; i++ {
		z[i] = new(Bar)
	}
	for i := 0; i < n; i++ {
		z[i] = nil
	}
}

func slabAllocate(n int, c *slabgo.Cache) {
	z := make([]*Bar, n)
	for i := 0; i < n; i++ {
		z[i] = c.Alloc().(*Bar)
	}
	for i := 0; i < n; i++ {
		c.Free(z[i])
		z[i] = nil
	}
}

func slabAllocatePtr(n int, c *slabgo.Cache) {
	z := make([]*Bar, n)
	for i := 0; i < n; i++ {
		z[i] = c.Alloc().(*Bar)
	}
	for i := 0; i < n; i++ {
		c.FreePtr(unsafe.Pointer(z[i]))
		z[i] = nil
	}
}

func BenchmarkBuiltin1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builtinAllocate(1000)
	}
}

func BenchmarkSlab1000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocate(1000, c)
	}
}

func BenchmarkSlabPtr1000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocatePtr(1000, c)
	}
}

func BenchmarkBuiltin10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builtinAllocate(10000)
	}
}

func BenchmarkSlab10000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocate(10000, c)
	}
}

func BenchmarkSlabPtr10000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocatePtr(10000, c)
	}
}

func BenchmarkBuiltin100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builtinAllocate(100000)
	}
}

func BenchmarkSlab100000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocate(100000, c)
	}
}

func BenchmarkSlabPtr100000(b *testing.B) {
	var a Bar
	c := slabgo.NewCacheSimple(a)
	for i := 0; i < b.N; i++ {
		slabAllocatePtr(100000, c)
	}
}
