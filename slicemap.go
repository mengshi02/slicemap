package tickdb

import (
	"hash/fnv"
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
	slots map[uint64]int
	nodes nodes
	freelist freelist
}

func New() SliceMapper {
	return &sliceMap{
		slots:    map[uint64]int{},
		nodes:    nodes{},
		freelist: freelist{},
	}
}

func (r *sliceMap) Set(key []byte, value []byte) {
	inode := r.freelist.allocate()
	if inode < 0 {
		inode = r.nodes.grow()
	}
	r.slots[sum64a(key)] = inode
	r.nodes.set(key, value, inode)
}

func (r *sliceMap) Get(key []byte) (b []byte, ok bool) {
	inode, ok := r.slots[sum64a(key)]
	if ok {
		return r.nodes[inode].value, ok
	}
	return
}

func (r *sliceMap) Has(key []byte) bool {
	_, ok := r.slots[sum64a(key)]
	return ok
}

func (r *sliceMap) ForRange(fn func(k, v []byte)) {
	for _, node := range r.nodes {
		fn(node.key, node.value)
	}
}

func (r *sliceMap) Del(key []byte) {
	k := sum64a(key)
	v, ok := r.slots[k]
	if !ok {
		return
	}
	r.nodes.del(v)
	delete(r.slots, k)
	r.freelist.reclamation(v)
}

func (r *sliceMap) Size() int {
	return len(r.slots)
}

func (r *sliceMap) Free() {
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
	return len(elems)-1
}

func (r *nodes) set(k, v []byte, inode int) {
	elems := *r
	elems[inode].key = append(elems[inode].key, k...)
	elems[inode].value = append(elems[inode].value, v...)
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
	key    []byte
	value  []byte
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
		*r = fl[:1]
		return n
	}
	return -1
}

func (r *freelist) reclamation(i int) {
	fl := *r
	if cap(fl) > len(fl) {
		fl[len(fl)+1] = i
		return
	}
	fl = append(fl, i)
	*r = fl
}

func (r *freelist) free() {
	fl := *r
	*r = fl[:0]
}

func sum64a(b []byte) uint64 {
	var h = fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}
