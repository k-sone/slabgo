package slabgo

import (
	"reflect"
	"sort"
	"unsafe"
)

var ntzMatrix [256]byte // The number of training zero

func buildNtzMatrix() {
	for i := 0; i < len(ntzMatrix); i++ {
		b := byte(i)
		b = ^b & (b + 1)
		b -= 1
		b = b&0x55 + (b>>1)&0x55
		b = b&0x33 + (b>>2)&0x33
		b = b&0x0f + (b>>4)&0xff
		ntzMatrix[i] = b
	}
}

// Constructor is called when a new slab is created.
// `objp` is a pointer of each new object.
type Constructor func(objp interface{})

// Destructor is called when a slab is destroyed.
// `objp` is a pointer of each object.
type Destructor func(objp interface{})

// Grower is called when there is not free slab.
// return an increment number of slabs.
type Grower func(s *CacheStats) int

// Reaper is called when there is free slab.
// return a decrement number of slabs.
type Reaper func(s *CacheStats) int

// Default implementation of grower.
var DefaultGrower Grower = func(s *CacheStats) int {
	if t := s.TotalSlabs; t == 0 {
		return 1
	} else if t < 32 {
		return t
	} else {
		return t / 4
	}
}

// Default implementation of reaper.
// slab is never freed.
var DefaultReaper Reaper = func(s *CacheStats) int {
	return 0
}

// Options for creating a Cache
type CacheOptions struct {
	ObjLen      int // length of object array within a slab, this is must be multiple of 8
	Grower      Grower
	Reaper      Reaper
	Constructor Constructor
	Destructor  Destructor
}

// Cache statistics
type CacheStats struct {
	TotalSlabs     int    // number of slab
	InuseSlabs     int    // number of slab in use
	TotalObjs      int    // number of object
	InuseObjs      int    // number of object in use
	Allocs         uint64 // number of allocs
	Frees          uint64 // number of frees
	CacheSize      uint64 // bytes of cache size
	CacheSizeInuse uint64 // bytes of cache size in use
}

// Storage for a specific type of object
type Cache struct {
	full      slabs // all objects within a slab marked as used
	partial   slabs // slab consists of both used and free objects
	empty     slabs // all objects within a slab marked as free
	objType   reflect.Type
	objLen    int
	inuseObjs int
	allocs    uint64
	frees     uint64
	grower    Grower
	reaper    Reaper
	ctor      Constructor
	dtor      Destructor
}

func (c *Cache) grow() int {
	var s CacheStats
	c.ReadStats(&s)
	num := c.grower(&s)

	if num < 0 {
		num = 0
	}
	for i := 0; i < num; i++ {
		(&c.empty).insert(newSlab(c.objType, c.objLen, c.ctor))
	}
	return num
}

func (c *Cache) reap() int {
	var s CacheStats
	c.ReadStats(&s)
	num := c.reaper(&s)

	if elen := len(c.empty); num > elen {
		num = elen
	}
	for i := 0; i < num; i++ {
		(&c.empty).pop(len(c.empty) - 1).destroy(c.dtor)
	}
	return num
}

// Allocate an object from cache.
// return a pointer of object.
func (c *Cache) Alloc() (obj interface{}) {
	if len(c.partial) == 0 {
		if len(c.empty) == 0 && c.grow() == 0 {
			// there is no available slab
			return
		}
		(&c.partial).insert((&c.empty).pop(0))
	}

	s := c.partial[0]
	obj = s.alloc()
	if s.total == s.inuse {
		(&c.full).insert((&c.partial).pop(0))
	}

	c.inuseObjs++
	c.allocs++
	return
}

// NOTE: This function is slow. It is recommended that `Cache.FreePtr` call be used instead.
//
// Return an object to cache.
// `objp` is a pointer of object.
func (c *Cache) Free(objp interface{}) bool {
	val := reflect.Indirect(reflect.ValueOf(objp))
	if !val.IsValid() || !val.CanAddr() {
		// invalid type
		return false
	}
	return c.free(val.UnsafeAddr())
}

// Return an object to cache.
// `objp` is a pointer of object.
func (c *Cache) FreePtr(objp unsafe.Pointer) bool {
	return c.free(uintptr(objp))
}

func (c *Cache) free(ptr uintptr) bool {
	// free from partial
	if i := c.partial.find(ptr); i > -1 {
		if s := c.partial[i]; s.free(ptr) {
			if s.inuse == 0 {
				(&c.empty).insert((&c.partial).pop(i))
				c.reap()
			}
			c.inuseObjs--
			c.frees++
			return true
		}
	}

	// free from full
	if i := c.full.find(ptr); i > -1 {
		if s := c.full[i]; s.free(ptr) {
			if s.inuse == 0 {
				(&c.empty).insert((&c.full).pop(i))
				c.reap()
			} else {
				(&c.partial).insert((&c.full).pop(i))
			}
			c.inuseObjs--
			c.frees++
			return true
		}
	}

	// not found
	return false
}

// Explicitly destroy a cache
func (c *Cache) Destroy() {
	for i := len(c.full) - 1; i > -1; i-- {
		(&c.full).pop(i).destroy(c.dtor)
	}
	for i := len(c.partial) - 1; i > -1; i-- {
		(&c.partial).pop(i).destroy(c.dtor)
	}
	for i := len(c.empty) - 1; i > -1; i-- {
		(&c.empty).pop(i).destroy(c.dtor)
	}
	c.inuseObjs = 0
	c.allocs = 0
	c.frees = 0
}

// Return type of object
func (c *Cache) ObjectType() reflect.Type {
	return c.objType
}

// Return length of object array within a slab
func (c *Cache) ObjectLen() int {
	return c.objLen
}

// Populates `s` with cache statistics
func (c *Cache) ReadStats(s *CacheStats) {
	objSize := uint64(c.objType.Size())

	s.TotalSlabs = len(c.full) + len(c.partial) + len(c.empty)
	s.InuseSlabs = len(c.full) + len(c.partial)
	s.TotalObjs = s.TotalSlabs * c.objLen
	s.InuseObjs = c.inuseObjs
	s.Allocs = c.allocs
	s.Frees = c.frees
	s.CacheSize = objSize * uint64(s.TotalObjs)
	s.CacheSizeInuse = objSize * uint64(s.InuseObjs)
}

// Create a Cache with options.
func NewCache(obj interface{}, opts CacheOptions) *Cache {
	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return nil
	}
	objtype := val.Type()
	objsize := objtype.Size()
	if objsize < 1 {
		return nil
	}

	objlen := opts.ObjLen
	if objlen == 0 {
		objlen = 256
	} else if mod := objlen & 0x07; mod != 0 {
		objlen += 8 - mod
	}

	grower := opts.Grower
	if grower == nil {
		grower = DefaultGrower
	}

	reaper := opts.Reaper
	if reaper == nil {
		reaper = DefaultReaper
	}

	return &Cache{
		objType: objtype,
		objLen:  objlen,
		grower:  grower,
		reaper:  reaper,
		ctor:    opts.Constructor,
		dtor:    opts.Destructor,
	}
}

// Create a Cache simply.
func NewCacheSimple(obj interface{}) *Cache {
	return NewCache(obj, CacheOptions{})
}

type slab struct {
	total   int     // number of all objects
	inuse   int     // number of objects that are inuse
	first   int     // index of a first object that are unused
	objsize uintptr // number of object size
	smem    uintptr // starting address of object array within slab
	emem    uintptr // end address of object array within slab
	bufctl  []byte  // bits of use state(0: unused, 1: inuse)
	chunk   []interface{}
}

func (s *slab) alloc() (obj interface{}) {
	obj = s.chunk[s.first]
	s.bufctl[s.first>>3] |= (1 << uint(s.first&0x7))
	s.inuse++

	// find a next object that are unused
	if s.total > s.inuse {
		for i := s.first >> 3; i < len(s.bufctl); i++ {
			if s.bufctl[i] != 0xff {
				s.first = i<<3 + int(ntzMatrix[s.bufctl[i]])
				return
			}
		}
	}

	// set out of range
	s.first = len(s.bufctl) << 3
	return
}

func (s *slab) free(optr uintptr) bool {
	if optr > s.emem {
		return false
	}

	// search target object
	lptr := optr - s.smem
	iptr := lptr / s.objsize
	if lptr != iptr*s.objsize {
		return false
	}

	i := int(iptr)
	s.bufctl[i>>3] ^= (1 << uint(i&0x7))
	s.inuse--
	if s.first > i {
		s.first = i
	}
	return true
}

func (s *slab) destroy(dtor Destructor) {
	if dtor != nil {
		for _, o := range s.chunk {
			dtor(o)
		}
	}
}

func newSlab(otype reflect.Type, size int, ctor Constructor) *slab {
	if mod := size & 0x07; size < 1 || mod != 0 {
		return nil
	}

	slice := reflect.MakeSlice(reflect.SliceOf(otype), size, size)
	chunk := make([]interface{}, size)
	for i := 0; i < size; i++ {
		chunk[i] = slice.Index(i).Addr().Interface()
	}

	if ctor != nil {
		for _, o := range chunk {
			ctor(o)
		}
	}

	return &slab{
		total:   size,
		inuse:   0,
		first:   0,
		objsize: otype.Size(),
		smem:    slice.Index(0).UnsafeAddr(),
		emem:    slice.Index(size - 1).UnsafeAddr(),
		bufctl:  make([]byte, size>>3),
		chunk:   chunk,
	}
}

type slabs []*slab

func (s slabs) find(p uintptr) int {
	return sort.Search(len(s), func(i int) bool { return s[i].smem > p }) - 1
}

func (s *slabs) insert(o *slab) {
	i := s.find(o.smem) + 1
	*s = append((*s), nil)
	copy((*s)[i+1:], (*s)[i:])
	(*s)[i] = o
}

func (s *slabs) pop(i int) (o *slab) {
	o = (*s)[i]
	copy((*s)[i:], (*s)[i+1:])
	(*s)[len(*s)-1] = nil
	*s = (*s)[:len(*s)-1]
	return
}

func init() {
	buildNtzMatrix()
}
