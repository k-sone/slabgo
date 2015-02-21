slabgo
======

Slab like memory cache for golang.

Overview
========

特定オブジェクトの確保と廃棄を繰り返すアプリケーションで, ガベージコレクションを減らすことにより
パフォーマンスを改善します.

オブジェクトキャッシングの仕組みは Slab Allocator を参考にしています.

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

- `Cache` は複数の `slab` を持ち, `slab` はオブジェクト配列を持ちます
- `Cache.Alloc` が実行されると未使用オブジェクトの参照を返します
    * 未使用オブジェクトが存在しない場合は, `slab` を作成します
- `Cache.Free` または `Cache.FreePtr` が実行されると, 指定したオブジェクトを未使用に設定します
    * オブジェクト配列が全て未使用の `slab` があれば削除します

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

// キャッシュ生成
var foo Foo
cache := slabgo.NewCacheSimple(foo)

// オブジェクト割り当て
obj := cache.Alloc().(*Foo)

// オブジェクト初期化 (前回利用時のデータが残っている可能性があるため)
obj.name = ""
obj.count = 0
obj.next = nil

// オブジェクト利用
...

// オブジェクト解放
cache.FreePtr(unsafe.Pointer(obj))

// 解放済みオブジェクトは利用禁止
obj = nil
```

License
-------

MIT
