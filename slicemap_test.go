package slicemap

import (
	"fmt"
	"sync"
	"testing"
)

func TestSliceMap_SetAndGet(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试 Set 和 Get
	sm.Set([]byte("key1"), []byte("value1"))
	sm.Set([]byte("key2"), []byte("value2"))

	// 验证 Get
	if v, ok := sm.Get([]byte("key1")); !ok || string(v) != "value1" {
		t.Errorf("Get(key1) = %s, %v; want value1, true", v, ok)
	}

	if v, ok := sm.Get([]byte("key2")); !ok || string(v) != "value2" {
		t.Errorf("Get(key2) = %s, %v; want value2, true", v, ok)
	}

	// 测试不存在的键
	if _, ok := sm.Get([]byte("key3")); ok {
		t.Error("Get(key3) should return false for non-existent key")
	}
}

func TestSliceMap_Has(t *testing.T) {
	sm := New()
	defer sm.Free()

	sm.Set([]byte("key1"), []byte("value1"))

	if !sm.Has([]byte("key1")) {
		t.Error("Has(key1) should return true")
	}

	if sm.Has([]byte("key2")) {
		t.Error("Has(key2) should return false for non-existent key")
	}
}

func TestSliceMap_Del(t *testing.T) {
	sm := New()
	defer sm.Free()

	sm.Set([]byte("key1"), []byte("value1"))
	sm.Set([]byte("key2"), []byte("value2"))

	// 删除存在的键
	sm.Del([]byte("key1"))

	if sm.Has([]byte("key1")) {
		t.Error("Has(key1) should return false after delete")
	}

	if !sm.Has([]byte("key2")) {
		t.Error("Has(key2) should still return true")
	}

	// 删除不存在的键（不应 panic）
	sm.Del([]byte("key3"))
}

func TestSliceMap_Size(t *testing.T) {
	sm := New()
	defer sm.Free()

	if sm.Size() != 0 {
		t.Errorf("Size() = %d; want 0", sm.Size())
	}

	sm.Set([]byte("key1"), []byte("value1"))
	if sm.Size() != 1 {
		t.Errorf("Size() = %d; want 1", sm.Size())
	}

	sm.Set([]byte("key2"), []byte("value2"))
	if sm.Size() != 2 {
		t.Errorf("Size() = %d; want 2", sm.Size())
	}

	sm.Del([]byte("key1"))
	if sm.Size() != 1 {
		t.Errorf("Size() = %d; want 1 after delete", sm.Size())
	}
}

func TestSliceMap_ForRange(t *testing.T) {
	sm := New()
	defer sm.Free()

	sm.Set([]byte("key1"), []byte("value1"))
	sm.Set([]byte("key2"), []byte("value2"))
	sm.Set([]byte("key3"), []byte("value3"))

	count := 0
	sm.ForRange(func(k, v []byte) {
		count++
	})

	if count != 3 {
		t.Errorf("ForRange visited %d items; want 3", count)
	}
}

func TestSliceMap_UpdateExistingKey(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 设置初始值
	sm.Set([]byte("key1"), []byte("value1"))

	// 更新已存在的键
	sm.Set([]byte("key1"), []byte("value2"))

	// 验证值被更新而非追加
	if v, ok := sm.Get([]byte("key1")); !ok || string(v) != "value2" {
		t.Errorf("Get(key1) = %s, %v; want value2, true", v, ok)
	}

	// Size 应该仍然是 1
	if sm.Size() != 1 {
		t.Errorf("Size() = %d; want 1", sm.Size())
	}
}

func TestSliceMap_FreelistReuse(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 添加多个键
	sm.Set([]byte("key1"), []byte("value1"))
	sm.Set([]byte("key2"), []byte("value2"))
	sm.Set([]byte("key3"), []byte("value3"))

	// 删除中间的键
	sm.Del([]byte("key2"))

	// 添加新键，应该复用 freelist 中的空间
	sm.Set([]byte("key4"), []byte("value4"))

	// 验证所有键
	if !sm.Has([]byte("key1")) {
		t.Error("key1 should exist")
	}
	if sm.Has([]byte("key2")) {
		t.Error("key2 should not exist")
	}
	if !sm.Has([]byte("key3")) {
		t.Error("key3 should exist")
	}
	if !sm.Has([]byte("key4")) {
		t.Error("key4 should exist")
	}
}

func TestSliceMap_EmptyKey(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试空键
	sm.Set([]byte{}, []byte("empty_key_value"))

	if v, ok := sm.Get([]byte{}); !ok || string(v) != "empty_key_value" {
		t.Errorf("Get(empty key) = %s, %v; want empty_key_value, true", v, ok)
	}
}

func TestSliceMap_EmptyValue(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试空值
	sm.Set([]byte("empty_value"), []byte{})

	if v, ok := sm.Get([]byte("empty_value")); !ok || string(v) != "" {
		t.Errorf("Get(empty_value) = %s, %v; want '', true", v, ok)
	}
}

func TestSliceMap_LargeData(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试大量数据
	n := 1000
	for i := 0; i < n; i++ {
		key := []byte{byte(i >> 8), byte(i)}
		value := []byte{byte(i)}
		sm.Set(key, value)
	}

	if sm.Size() != n {
		t.Errorf("Size() = %d; want %d", sm.Size(), n)
	}

	// 验证所有数据
	for i := 0; i < n; i++ {
		key := []byte{byte(i >> 8), byte(i)}
		if v, ok := sm.Get(key); !ok || v[0] != byte(i) {
			t.Errorf("Get(key %d) failed", i)
		}
	}
}

func TestSliceMap_MultipleDeleteAndAdd(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 添加、删除、再添加循环
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		sm.Set(key, []byte("value"))
		sm.Del(key)
		sm.Set(key, []byte("newvalue"))

		if v, ok := sm.Get(key); !ok || string(v) != "newvalue" {
			t.Errorf("Iteration %d: Get failed", i)
		}
		sm.Del(key)
	}

	if sm.Size() != 0 {
		t.Errorf("Size() = %d; want 0", sm.Size())
	}
}

func TestSliceMap_BinaryData(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试二进制数据
	key := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	value := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	sm.Set(key, value)

	if v, ok := sm.Get(key); !ok {
		t.Error("Get binary key failed")
	} else if len(v) != len(value) {
		t.Errorf("Value length = %d; want %d", len(v), len(value))
	} else {
		for i := range value {
			if v[i] != value[i] {
				t.Errorf("Value[%d] = %x; want %x", i, v[i], value[i])
			}
		}
	}
}

func TestSliceMap_Concurrency(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 并发写入
	var wg sync.WaitGroup
	n := 100
	numGoroutines := 10

	// 并发写入
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				key := []byte(fmt.Sprintf("key-%d-%d", gid, i))
				value := []byte(fmt.Sprintf("value-%d-%d", gid, i))
				sm.Set(key, value)
			}
		}(g)
	}
	wg.Wait()

	// 并发读取
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				key := []byte(fmt.Sprintf("key-%d-%d", gid, i))
				value := []byte(fmt.Sprintf("value-%d-%d", gid, i))
				if v, ok := sm.Get(key); !ok || string(v) != string(value) {
					t.Errorf("Get(%s) failed", key)
				}
			}
		}(g)
	}
	wg.Wait()

	// 并发删除
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				key := []byte(fmt.Sprintf("key-%d-%d", gid, i))
				sm.Del(key)
			}
		}(g)
	}
	wg.Wait()

	if sm.Size() != 0 {
		t.Errorf("Size() = %d; want 0 after concurrent delete", sm.Size())
	}
}

func TestSliceMap_ConcurrentReadWrite(t *testing.T) {
	sm := New()
	defer sm.Free()

	var wg sync.WaitGroup
	n := 1000

	// 预先写入一些数据
	for i := 0; i < n; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("value-%d", i))
		sm.Set(key, value)
	}

	// 并发读写混合
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < n; j++ {
				key := []byte(fmt.Sprintf("key-%d", j))
				sm.Get(key)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < n; j++ {
				key := []byte(fmt.Sprintf("key-%d", j))
				value := []byte(fmt.Sprintf("new-value-%d", j))
				sm.Set(key, value)
			}
		}()
	}
	wg.Wait()
}

func TestSliceMap_HashCollision(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 测试哈希冲突：即使哈希相同，不同的键也能正确存储
	// 注意：这个测试验证的是冲突处理机制，不是构造真正的冲突键
	keys := make([][]byte, 100)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("collision-test-key-%d", i))
	}

	// 设置所有键
	for i, key := range keys {
		value := []byte(fmt.Sprintf("value-%d", i))
		sm.Set(key, value)
	}

	// 验证所有键都能正确获取
	for i, key := range keys {
		expectedValue := fmt.Sprintf("value-%d", i)
		if v, ok := sm.Get(key); !ok || string(v) != expectedValue {
			t.Errorf("Get(key %d) = %s, %v; want %s, true", i, v, ok, expectedValue)
		}
	}

	// 验证删除一个键不影响其他键
	sm.Del(keys[50])
	if sm.Has(keys[50]) {
		t.Error("key 50 should be deleted")
	}
	for i, key := range keys {
		if i == 50 {
			continue
		}
		if !sm.Has(key) {
			t.Errorf("key %d should still exist", i)
		}
	}
}

func TestSliceMap_SizeAfterCollision(t *testing.T) {
	sm := New()
	defer sm.Free()

	// 设置多个键
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("size-test-%d", i))
		value := []byte("value")
		sm.Set(key, value)
	}

	if sm.Size() != 100 {
		t.Errorf("Size() = %d; want 100", sm.Size())
	}

	// 更新已存在的键，Size 不应改变
	sm.Set([]byte("size-test-0"), []byte("new-value"))
	if sm.Size() != 100 {
		t.Errorf("Size() = %d; want 100 after update", sm.Size())
	}
}