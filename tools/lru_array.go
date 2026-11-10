package tools

import "sync"

// LRUArray 实现一个固定容量的 LRU（最近最少使用）数组
// 新元素添加到末尾，超出容量时移除最旧的元素（头部）
type LRUArray struct {
	mu    sync.Mutex
	Cap   int
	array []any
}

// NewLRUArray 创建指定容量的 LRU 数组
func NewLRUArray(cap int) *LRUArray {
	if cap <= 0 {
		cap = 10
	}

	return &LRUArray{
		Cap:   cap,
		array: make([]any, 0, cap), // 长度为0，容量为cap
	}
}

// Add 添加元素到 LRU 数组
// - 如果数组未满，直接追加
// - 如果已满，移除最旧的元素（头部），在末尾添加新元素
func (la *LRUArray) Add(a any) {
	la.mu.Lock()
	defer la.mu.Unlock()

	if len(la.array) < la.Cap {
		// 未满，直接追加
		la.array = append(la.array, a)
	} else {
		// 已满，
		copy(la.array, la.array[1:])
		la.array[la.Cap-1] = a
	}
}

// Find 和 Replace都不会影响元素位置

// Find 返回第一个满足条件的元素。
func (la *LRUArray) Find(match func(a any) bool) (any, bool) {
	la.mu.Lock()
	defer la.mu.Unlock()

	for _, a := range la.array {
		if match(a) {
			return a, true
		}
	}
	return nil, false
}

// Replace 用新元素替换第一个满足条件的元素，返回是否命中。
func (la *LRUArray) Replace(match func(a any) bool, a any) bool {
	la.mu.Lock()
	defer la.mu.Unlock()

	for i, v := range la.array {
		if match(v) {
			la.array[i] = a
			return true
		}
	}
	return false
}

// Clear 清空数组
func (la *LRUArray) Clear() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.array = la.array[:0]
}

// Len 返回当前元素个数
func (la *LRUArray) Len() int {
	la.mu.Lock()
	defer la.mu.Unlock()
	return len(la.array)
}
