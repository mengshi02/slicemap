# SliceMap

SliceMap 是一个基于 Go 语言实现的高性能、并发安全的 Map 数据结构，结合了原生 Map 的查找性能和 Slice 的遍历特性。

## 特性

- **并发安全**：使用 `sync.RWMutex` 实现读写锁，支持高并发场景
- **哈希冲突处理**：使用链地址法（separate chaining）处理哈希冲突，避免数据丢失
- **混合存储**：使用 `map[uint64][]int` 存储索引链，`slice` 存储实际数据
- **接口支持**：实现了通用的 Map 操作接口 (`Set`, `Get`, `Has`, `Del`, `ForRange`, `Size`, `Free`)
- **内存回收**：使用 freelist 机制回收已删除节点的空间
- **字节切片键值**：键和值都使用 `[]byte` 类型，适用于二进制数据存储

## 安装

```bash
go get github.com/mengshi02/slicemap
```

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/mengshi02/slicemap"
)

func main() {
    // 创建一个新的 SliceMap
    sm := slicemap.New()

    // 设置键值对
    sm.Set([]byte("name"), []byte("Alice"))
    sm.Set([]byte("age"), []byte("25"))

    // 获取值
    if value, ok := sm.Get([]byte("name")); ok {
        fmt.Printf("name: %s\n", value)
    }

    // 检查键是否存在
    if sm.Has([]byte("age")) {
        fmt.Println("age exists")
    }

    // 获取大小
    fmt.Printf("size: %d\n", sm.Size())

    // 遍历所有键值对
    sm.ForRange(func(k, v []byte) {
        fmt.Printf("%s: %s\n", k, v)
    })

    // 删除键
    sm.Del([]byte("age"))

    // 释放资源
    sm.Free()
}
```

## API 文档

### SliceMapper 接口

```go
type SliceMapper interface {
    Set(k, v []byte)                    // 设置键值对
    Get(k []byte) ([]byte, bool)        // 获取值，返回值和是否存在
    Has(key []byte) bool                // 检查键是否存在
    Del(k []byte)                       // 删除键值对
    ForRange(fn func(k, v []byte))      // 遍历所有键值对
    Size() int                          // 返回键值对数量
    Free()                              // 释放资源
}
```

## 实现细节

### 数据结构

SliceMap 由以下几部分组成：

1. **mu (sync.RWMutex)**：读写锁，保证并发安全
2. **slots (map[uint64][]int)**：存储哈希值到节点索引列表的映射，支持哈希冲突处理
3. **nodes ([]node)**：存储实际的键值对数据
4. **freelist ([]int)**：存储已删除节点的索引，用于内存复用

### 并发安全

- 使用 `sync.RWMutex` 实现读写分离锁
- 读操作（Get、Has、Size、ForRange）使用 `RLock()`，支持并发读
- 写操作（Set、Del、Free）使用 `Lock()`，保证写安全

### 哈希冲突处理

- 使用链地址法（separate chaining）处理冲突
- 每个哈希值对应一个索引列表 `[]int`
- 查找时遍历索引列表，比较实际键值
- 确保不同键即使哈希相同也能正确存储和获取

### 内存管理

- 当删除键值对时，节点索引会被添加到 freelist 中
- 新增键值对时，优先从 freelist 分配空间
- 如果 freelist 为空，则扩展 nodes slice

## 性能特性

| 操作 | 时间复杂度 | 说明 |
|------|-----------|------|
| Set | O(1) 平均 | 最坏 O(n) 当大量哈希冲突 |
| Get | O(1) 平均 | 最坏 O(n) 当大量哈希冲突 |
| Has | O(1) 平均 | 最坏 O(n) 当大量哈希冲突 |
| Del | O(1) 平均 | 最坏 O(n) 当大量哈希冲突 |
| ForRange | O(n) | 顺序遍历所有有效节点 |
| Size | O(1) | 返回 slots 大小 |

## 适用场景

**推荐场景：**
1. **高并发缓存** - 读写锁支持并发读写
2. **需要顺序遍历的场景** - 遍历性能优于原生 map
3. **二进制数据存储** - 网络协议、序列化数据
4. **频繁增删** - freelist 复用减少 GC 压力

**不推荐场景：**
1. **需要类型安全** - 仅支持 `[]byte` 键值
2. **需要有序遍历** - 遍历顺序为插入顺序，非排序

## License

MIT License