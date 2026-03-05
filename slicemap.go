package slicemap

import (
	"hash/fnv"
	"sync"
)

type SliceMapper interface {
	Set(k, v []byte)
	Get(k []byte) ([]byte, bool)
	Has(key []byte) bool
	Del(k []byte)
	ForRange(fn func(k, v []byte))
	Size() int
	Free()
}

type sliceMap struct {
	mu     sync.RWMutex
	slots  map[uint64][]int // 哈希值 -> 节点索引列表（处理冲突）
	nodes  nodes
	freelist freelist
}

func New() SliceMapper {
	return &sliceMap{
		slots:    map[uint64][]int{},
		nodes:    nodes{},
		freelist: freelist{},
	}
}

func (r *sliceMap) Set(key []byte, value []byte) {
	hash := sum64a(key)
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 检查是否存在相同键
	if indices, exists := r.slots[hash]; exists {
		for _, idx := range indices {
			if bytesEqual(r.nodes[idx].key, key) {
				// 找到相同键，更新值
				r.nodes.set(key, value, idx)
				return
			}
		}
		// 哈希冲突：添加新节点到索引列表
		inode := r.allocateNode()
		r.slots[hash] = append(indices, inode)
		r.nodes.set(key, value, inode)
		return
	}
	
	// 新键
	inode := r.allocateNode()
	r.slots[hash] = []int{inode}
	r.nodes.set(key, value, inode)
}

func (r *sliceMap) allocateNode() int {
	inode := r.freelist.allocate()
	if inode < 0 {
		inode = r.nodes.grow()
	}
	return inode
}

func (r *sliceMap) Get(key []byte) (b []byte, ok bool) {
	hash := sum64a(key)
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	indices, exists := r.slots[hash]
	if !exists {
		return nil, false
	}
	
	// 遍历冲突链查找匹配的键
	for _, idx := range indices {
		if bytesEqual(r.nodes[idx].key, key) {
			return r.nodes[idx].value, true
		}
	}
	
	return nil, false
}

func (r *sliceMap) Has(key []byte) bool {
	hash := sum64a(key)
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	indices, exists := r.slots[hash]
	if !exists {
		return false
	}
	
	// 遍历冲突链查找匹配的键
	for _, idx := range indices {
		if bytesEqual(r.nodes[idx].key, key) {
			return true
		}
	}
	
	return false
}

func (r *sliceMap) ForRange(fn func(k, v []byte)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, node := range r.nodes {
		if len(node.key) > 0 { // 只遍历有效节点
			fn(node.key, node.value)
		}
	}
}

func (r *sliceMap) Del(key []byte) {
	hash := sum64a(key)
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	indices, exists := r.slots[hash]
	if !exists {
		return
	}
	
	// 查找并删除匹配的键
	for i, idx := range indices {
		if bytesEqual(r.nodes[idx].key, key) {
			// 回收节点
			r.nodes.del(idx)
			r.freelist.reclamation(idx)
			
			// 从索引列表中移除
			if len(indices) == 1 {
				delete(r.slots, hash)
			} else {
				r.slots[hash] = append(indices[:i], indices[i+1:]...)
			}
			return
		}
	}
}

func (r *sliceMap) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.slots)
}

func (r *sliceMap) Free() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slots = nil
	r.nodes.free()
	r.freelist.free()
}

type nodes []node

func (r *nodes) grow() int {
	elems := *r
	if cap(elems) > len(elems) {
		elems = elems[:len(elems)+1]
	} else {
		elems = append(elems, node{})
	}
	*r = elems
	return len(elems) - 1
}

func (r *nodes) set(k, v []byte, inode int) {
	elems := *r
	// 重置后再设置，避免追加
	elems[inode].key = append(elems[inode].key[:0], k...)
	elems[inode].value = append(elems[inode].value[:0], v...)
	*r = elems
}

func (r *nodes) get(i int) node {
	elems := *r
	return elems[i]
}

func (r *nodes) del(i int) {
	elems := *r
	elems[i].reset()
}

func (r *nodes) free() {
	elems := *r
	*r = elems[:0]
}

type node struct {
	key   []byte
	value []byte
}

func (r *node) reset() {
	r.key = r.key[:0]
	r.value = r.value[:0]
}

type freelist []int

func (r *freelist) allocate() int {
	fl := *r
	if len(fl) > 0 {
		n := fl[0]
		*r = fl[1:]
		return n
	}
	return -1
}

func (r *freelist) reclamation(i int) {
	fl := *r
	fl = append(fl, i)
	*r = fl
}

func (r *freelist) free() {
	fl := *r
	*r = fl[:0]
}

func sum64a(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}

// bytesEqual 比较两个字节切片是否相等
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}