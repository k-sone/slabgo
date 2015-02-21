slabgo
======

Slab like memory cache for golang.

Overview
========

This library improves performance of application that repeat allocation and release
of a specific object by reducing garbage collection.

An object caching mechanism is referencing Slab Allocator.

```
 Cache
+---------------------------------+
|  slab list                      |
| +------+  +------+     +------+ |
| | slab |  | slab | ... | slab | |
| +------+  +------+     +------+ |
+---------------------------------+

 slab
+---------------------------+
|  object array             |
| +-----+-----+-----+-----+ |
| |  0  |  1  | ... | n-1 | | (n is length of object array within a slab)
| +-----+-----+-----+-----+ |
+---------------------------+
```

- `Cache` has list of `slab`, `slab` has an object array
- Return a pointer of unused object when you call `Cache.Alloc` method
    * Create `slab` into `Cache` if unused object not exists
- Mark to unused the specified object when you call `Cache.Free` or `Cache.FreePtr` method
    * Delete `slab` if there is `slab` that all objects marked as unused

Examples
--------

```go
import (
    "unsafe"

    "github.com/k-sone/slabgo"
)

type Foo struct {
    name  string
    count int64
    next  *Foo
}

...

// create cache
var foo Foo
cache := slabgo.NewCacheSimple(foo)

// allocate object
obj := cache.Alloc().(*Foo)

// initialize object (object may be recycled)
obj.name = ""
obj.count = 0
obj.next = nil

// use object
...

// release object
cache.FreePtr(unsafe.Pointer(obj))

// don't use object after release
obj = nil
```

License
-------

MIT
