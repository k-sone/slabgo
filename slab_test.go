package slabgo_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/k-sone/slabgo"
)

type Foo struct {
	name  string
	count int64
	next  *Foo
}

func counter(n *int) func(interface{}) {
	return func(objp interface{}) {
		if _, ok := objp.(*Foo); ok {
			(*n)++
		}
	}
}

func checkStats(t *testing.T, name string, c *slabgo.Cache, exp *slabgo.CacheStats) {
	var act slabgo.CacheStats
	c.ReadStats(&act)
	if act.TotalSlabs != exp.TotalSlabs {
		t.Errorf("%s - total slabs: expected [%d], actual [%d]", name, exp.TotalSlabs, act.TotalSlabs)
	}
	if act.InuseSlabs != exp.InuseSlabs {
		t.Errorf("%s - inuse slabs: expected [%d], actual [%d]", name, exp.InuseSlabs, act.InuseSlabs)
	}
	if act.TotalObjs != exp.TotalObjs {
		t.Errorf("%s - total objs: expected [%d], actual [%d]", name, exp.TotalObjs, act.TotalObjs)
	}
	if act.InuseObjs != exp.InuseObjs {
		t.Errorf("%s - inuse objs: expected [%d], actual [%d]", name, exp.InuseObjs, act.InuseObjs)
	}
	if act.Allocs != exp.Allocs {
		t.Errorf("%s - allocs: expected [%d], actual [%d]", name, exp.Allocs, act.Allocs)
	}
	if act.Frees != exp.Frees {
		t.Errorf("%s - fress: expected [%d], actual [%d]", name, exp.Frees, act.Frees)
	}
}

func checkGrow(t *testing.T, name string, act, exp int) {
	if act != exp {
		t.Errorf("%s - grow: expected [%d], actual [%d]", name, exp, act)
	}
}

func checkReap(t *testing.T, name string, act, exp int) {
	if act != exp {
		t.Errorf("%s - reap: expected [%d], actual [%d]", name, exp, act)
	}
}

func checkConstruct(t *testing.T, name string, act, exp int) {
	if act != exp {
		t.Errorf("%s - construct: expected [%d], actual [%d]", name, exp, act)
	}
}

func checkDestruct(t *testing.T, name string, act, exp int) {
	if act != exp {
		t.Errorf("%s - destruct: expected [%d], actual [%d]", name, exp, act)
	}
}

func TestSlabNew(t *testing.T) {
	var foo Foo

	expType := reflect.TypeOf(foo)
	objLen := 32
	grow := func(s *slabgo.CacheStats) int { return 1 }
	reap := func(s *slabgo.CacheStats) int { return 1 }

	cache := slabgo.NewCache(foo, slabgo.CacheOptions{
		ObjLen: objLen,
		Grower: grow,
		Reaper: reap,
	})
	if cache == nil {
		t.Error("NewCache() - failed")
	}
	if n := cache.ObjectType(); n != expType {
		t.Errorf("ObjectType() - expected [%s], actual [%s]", expType, n)
	}
	if n := cache.ObjectLen(); n != objLen {
		t.Errorf("ObjectLen() - expected [%d], actual [%d]", objLen, n)
	}

	cache = slabgo.NewCacheSimple(foo)
	if cache == nil {
		t.Error("NewCacheSimple() - failed")
	}
	if n := cache.ObjectType(); n != expType {
		t.Errorf("ObjectType() - expected [%s], actual [%s]", expType, n)
	}
	if n := cache.ObjectLen(); n == 0 || n%8 != 0 {
		t.Errorf("ObjectLen() - expected [%d], actual [%d]", objLen, n)
	}

	cache = slabgo.NewCacheSimple(nil)
	if cache != nil {
		t.Error("NewCacheSimple() - nil failed")
	}
}

func TestSlabAlloc(t *testing.T) {
	var foo Foo
	var foos []*Foo
	var stats slabgo.CacheStats

	objLen := 32
	var gnum, rnum, cnum, dnum int

	cache := slabgo.NewCache(foo, slabgo.CacheOptions{
		ObjLen:      objLen,
		Grower:      func(s *slabgo.CacheStats) int { gnum++; return 1 },
		Reaper:      func(s *slabgo.CacheStats) int { rnum++; return 1 },
		Constructor: counter(&cnum),
		Destructor:  counter(&dnum),
	})

	// first alloc
	o := cache.Alloc()
	if o == nil {
		t.Error("Alloc() - failed")
	}
	if f, ok := o.(*Foo); ok {
		foos = append(foos, f)
	} else {
		t.Errorf("Alloc() - invalid type %s", o)
	}

	name := "Alloc() 1st"
	stats.TotalSlabs = 1
	stats.InuseSlabs = 1
	stats.TotalObjs = objLen
	stats.InuseObjs = 1
	stats.Allocs = 1
	stats.Frees = 0
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 1)
	checkReap(t, name, rnum, 0)
	checkConstruct(t, name, cnum, objLen)
	checkDestruct(t, name, dnum, 0)

	// all objects within a slab marked as used
	for i := 0; i < objLen-1; i++ {
		if f, ok := cache.Alloc().(*Foo); ok {
			foos = append(foos, f)
		}
	}

	name = "Alloc() 2nd"
	stats.TotalSlabs = 1
	stats.InuseSlabs = 1
	stats.TotalObjs = objLen
	stats.InuseObjs = 32
	stats.Allocs = 32
	stats.Frees = 0
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 1)
	checkReap(t, name, rnum, 0)
	checkConstruct(t, name, cnum, objLen)
	checkDestruct(t, name, dnum, 0)

	// add two slabs
	for i := 0; i < objLen*2; i++ {
		if f, ok := cache.Alloc().(*Foo); ok {
			foos = append(foos, f)
		}
	}

	name = "Alloc() 3rd"
	stats.TotalSlabs = 3
	stats.InuseSlabs = 3
	stats.TotalObjs = objLen * 3
	stats.InuseObjs = 32 * 3
	stats.Allocs = 32 * 3
	stats.Frees = 0
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 0)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, 0)
}

func TestSlabFree(t *testing.T) {
	var foo Foo
	var foos []*Foo
	var stats slabgo.CacheStats

	objLen := 64
	var gnum, rnum, cnum, dnum int

	cache := slabgo.NewCache(foo, slabgo.CacheOptions{
		ObjLen:      objLen,
		Grower:      func(s *slabgo.CacheStats) int { gnum++; return 1 },
		Reaper:      func(s *slabgo.CacheStats) int { rnum++; return 1 },
		Constructor: counter(&cnum),
		Destructor:  counter(&dnum),
	})

	for i := 0; i < objLen*3; i++ {
		if f, ok := cache.Alloc().(*Foo); ok {
			foos = append(foos, f)
		}
	}

	if cache.Free(nil) {
		t.Error("Free() - nil")
	}
	if cache.Free("test") {
		t.Error("Free() - invalid type")
	}
	if cache.Free(&foo) {
		t.Error("Free() - not allocated")
	}

	// first free
	if !cache.Free(foos[0]) {
		t.Error("Free() 1st - failed")
	}
	name := "Free() 1st"
	stats.TotalSlabs = 3
	stats.InuseSlabs = 3
	stats.TotalObjs = objLen * 3
	stats.InuseObjs = objLen*3 - 1
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = 1
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 0)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, 0)

	// free objects within last slab
	for i, o := range foos[objLen*2:] {
		if !cache.Free(o) {
			t.Errorf("Free() 2nd - failed at %d", i)
		}
	}

	name = "Free() 2nd"
	stats.TotalSlabs = 2
	stats.InuseSlabs = 2
	stats.TotalObjs = objLen * 2
	stats.InuseObjs = objLen*2 - 1
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen + 1)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 1)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen)

	// free objects within first slab
	for i, o := range foos[1:objLen] {
		if !cache.Free(o) {
			t.Errorf("Free() 3rd - failed at %d", i)
		}
	}

	name = "Free() 3rd"
	stats.TotalSlabs = 1
	stats.InuseSlabs = 1
	stats.TotalObjs = objLen
	stats.InuseObjs = objLen
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen * 2)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 2)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen*2)

	// free all objects
	for i, o := range foos[objLen : objLen*2] {
		if !cache.Free(o) {
			t.Errorf("Free() 4th - failed at %d", i)
		}
	}

	name = "Free() 4th"
	stats.TotalSlabs = 0
	stats.InuseSlabs = 0
	stats.TotalObjs = 0
	stats.InuseObjs = 0
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen * 3)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 3)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen*3)

	if cache.Free(foos[0]) {
		t.Errorf("Free() - double free")
	}
}

func TestSlabFreePtr(t *testing.T) {
	var foo Foo
	var foos []*Foo
	var stats slabgo.CacheStats

	objLen := 64
	var gnum, rnum, cnum, dnum int

	cache := slabgo.NewCache(foo, slabgo.CacheOptions{
		ObjLen:      objLen,
		Grower:      func(s *slabgo.CacheStats) int { gnum++; return 1 },
		Reaper:      func(s *slabgo.CacheStats) int { rnum++; return 1 },
		Constructor: counter(&cnum),
		Destructor:  counter(&dnum),
	})

	for i := 0; i < objLen*3; i++ {
		if f, ok := cache.Alloc().(*Foo); ok {
			foos = append(foos, f)
		}
	}

	if cache.FreePtr(unsafe.Pointer(nil)) {
		t.Error("FreePtr() - nil")
	}
	s := "test"
	if cache.FreePtr(unsafe.Pointer(&s)) {
		t.Error("FreePtr() - invalid type")
	}
	if cache.FreePtr(unsafe.Pointer(&foo)) {
		t.Error("FreePtr() - not allocated")
	}

	// first free
	if !cache.FreePtr(unsafe.Pointer(foos[0])) {
		t.Error("FreePtr() 1st - failed")
	}
	name := "FreePtr() 1st"
	stats.TotalSlabs = 3
	stats.InuseSlabs = 3
	stats.TotalObjs = objLen * 3
	stats.InuseObjs = objLen*3 - 1
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = 1
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 0)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, 0)

	// free objects within last slab
	for i, o := range foos[objLen*2:] {
		if !cache.FreePtr(unsafe.Pointer(o)) {
			t.Errorf("FreePtr() 2nd - failed at %d", i)
		}
	}

	name = "FreePtr() 2nd"
	stats.TotalSlabs = 2
	stats.InuseSlabs = 2
	stats.TotalObjs = objLen * 2
	stats.InuseObjs = objLen*2 - 1
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen + 1)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 1)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen)

	// free objects within first slab
	for i, o := range foos[1:objLen] {
		if !cache.FreePtr(unsafe.Pointer(o)) {
			t.Errorf("FreePtr() 3rd - failed at %d", i)
		}
	}

	name = "FreePtr() 3rd"
	stats.TotalSlabs = 1
	stats.InuseSlabs = 1
	stats.TotalObjs = objLen
	stats.InuseObjs = objLen
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen * 2)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 2)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen*2)

	// free all objects
	for i, o := range foos[objLen : objLen*2] {
		if !cache.FreePtr(unsafe.Pointer(o)) {
			t.Errorf("FreePtr() 4th - failed at %d", i)
		}
	}

	name = "FreePtr() 4th"
	stats.TotalSlabs = 0
	stats.InuseSlabs = 0
	stats.TotalObjs = 0
	stats.InuseObjs = 0
	stats.Allocs = uint64(objLen * 3)
	stats.Frees = uint64(objLen * 3)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 3)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, objLen*3)

	if cache.FreePtr(unsafe.Pointer(foos[0])) {
		t.Errorf("FreePtr() - double free")
	}
}

func TestSlabReAllocate(t *testing.T) {
	var foo Foo
	var foos []*Foo

	objLen := 32
	var gnum, rnum, cnum, dnum int
	var stats slabgo.CacheStats

	cache := slabgo.NewCache(foo, slabgo.CacheOptions{
		ObjLen:      objLen,
		Grower:      func(s *slabgo.CacheStats) int { gnum++; return 1 },
		Reaper:      func(s *slabgo.CacheStats) int { rnum++; return 0 },
		Constructor: counter(&cnum),
		Destructor:  counter(&dnum),
	})

	for i := 0; i < objLen*3; i++ {
		if f, ok := cache.Alloc().(*Foo); ok {
			foos = append(foos, f)
		}
	}

	foos[0].name = "aaa"
	foos[32].name = "bbb"
	foos[64].name = "ccc"
	cache.Free(foos[0])
	cache.Free(foos[32])
	cache.Free(foos[64])

	if f := cache.Alloc().(*Foo); f.name != "aaa" {
		t.Errorf("ReAllocate - failed at 0")
	}
	if f := cache.Alloc().(*Foo); f.name != "bbb" {
		t.Errorf("ReAllocate - failed at 32")
	}
	if f := cache.Alloc().(*Foo); f.name != "ccc" {
		t.Errorf("ReAllocate - failed at 64")
	}

	for i := 1; i <= objLen; i++ {
		cache.Free(foos[len(foos)-i])
	}
	cache.Alloc()

	name := "reallocate last free slab"
	stats.TotalSlabs = 3
	stats.InuseSlabs = 3
	stats.TotalObjs = objLen * 3
	stats.InuseObjs = objLen*2 + 1
	stats.Allocs = uint64(objLen*3 + 3 + 1)
	stats.Frees = uint64(objLen + 3)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 1)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, 0)

	for i := 0; i < objLen; i++ {
		cache.Free(foos[objLen+i])
	}
	cache.Alloc()

	name = "reallocate middle free slab"
	stats.TotalSlabs = 3
	stats.InuseSlabs = 2
	stats.TotalObjs = objLen * 3
	stats.InuseObjs = objLen + 1 + 1
	stats.Allocs = uint64(objLen*3 + 3 + 1 + 1)
	stats.Frees = uint64(objLen*2 + 3)
	checkStats(t, name, cache, &stats)
	checkGrow(t, name, gnum, 3)
	checkReap(t, name, rnum, 2)
	checkConstruct(t, name, cnum, objLen*3)
	checkDestruct(t, name, dnum, 0)
}
