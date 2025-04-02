// Package Queue
// @Author Clover
// @Data 2025/3/31 下午7:09:00
// @Desc 使用 Go 泛型实现的并发安全的切片队列
package Queue

import (
	"errors"
	"sync" // 导入 sync 包
)

// Queue 是一个泛型队列结构
type Queue[T any] struct {
	mu sync.Mutex // 互斥锁，用于保护队列的并发访问
	Q  []T        // 存储队列元素的切片
}

// NewQueue 创建并返回一个新的泛型队列实例
func NewQueue[T any]() *Queue[T] {
	// 初始化时也包含互斥锁
	return &Queue[T]{Q: make([]T, 0)}
}

// Enqueue 将一个元素添加到队列的末尾 (并发安全)
func (q *Queue[T]) Enqueue(item T) {
	q.mu.Lock()         // 获取锁
	defer q.mu.Unlock() // 保证函数退出时释放锁
	q.Q = append(q.Q, item)
}

// Dequeue 从队列的头部移除并返回一个元素 (并发安全)
// 如果队列为空，则返回 T 类型的零值和错误
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()         // 获取锁
	defer q.mu.Unlock() // 保证函数退出时释放锁

	if q.IsEmptyLocked() { // 使用内部的非锁定版本检查
		var zero T // 获取 T 类型的零值
		return zero, errors.New("queue is empty")
	}
	item := q.Q[0]
	q.Q = q.Q[1:] // 通过切片操作移除第一个元素
	return item, nil
}

// Peek 返回队列头部的元素但不移除它 (并发安全)
// 如果队列为空，则返回 T 类型的零值和错误
func (q *Queue[T]) Peek() (T, error) {
	q.mu.Lock()         // 获取锁
	defer q.mu.Unlock() // 保证函数退出时释放锁

	if q.IsEmptyLocked() { // 使用内部的非锁定版本检查
		var zero T
		return zero, errors.New("queue is empty")
	}
	return q.Q[0], nil
}

// IsEmpty 检查队列是否为空 (并发安全)
func (q *Queue[T]) IsEmpty() bool {
	q.mu.Lock()         // 获取锁
	defer q.mu.Unlock() // 保证函数退出时释放锁
	return len(q.Q) == 0
}

// IsEmptyLocked 是 IsEmpty 的内部非锁定版本
func (q *Queue[T]) IsEmptyLocked() bool {
	return len(q.Q) == 0
}

// Size 返回队列中元素的数量 (并发安全)
func (q *Queue[T]) Size() int {
	q.mu.Lock()         // 获取锁
	defer q.mu.Unlock() // 保证函数退出时释放锁
	return len(q.Q)
}
