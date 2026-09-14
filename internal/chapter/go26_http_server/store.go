package go26_http_server

import (
	"sort"
	"sync"
	"time"
)

// Task 是 REST 示例里的资源。
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Store 是并发安全的内存存储。
//
// now 让时间来源可注入：测试和演示换成固定时钟，输出就不再随运行时刻变化。
type Store struct {
	mu    sync.Mutex
	now   func() time.Time
	next  int
	items map[int]Task
}

// NewStore 创建使用真实时钟的存储。
func NewStore() *Store {
	return newStoreWithClock(time.Now)
}

// newStoreWithClock 允许注入时钟，便于写出确定性的测试和演示输出。
func newStoreWithClock(now func() time.Time) *Store {
	return &Store{
		now:   now,
		next:  1,
		items: make(map[int]Task),
	}
}

// List 按 ID 升序返回任务；done 非 nil 时按完成状态过滤，limit 大于 0 时截断。
func (s *Store) List(done *bool, limit int) []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, 0, len(s.items))
	for _, task := range s.items {
		if done != nil && task.Done != *done {
			continue
		}
		out = append(out, task)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Get 按 ID 查询单个任务。
func (s *Store) Get(id int) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.items[id]
	return task, ok
}

// Create 新建任务并返回写入后的结果。
func (s *Store) Create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := Task{
		ID:        s.next,
		Title:     title,
		CreatedAt: s.now().UTC(),
	}
	s.next++
	s.items[task.ID] = task
	return task
}

// Update 局部更新任务；nil 字段表示不改。返回更新后的任务以及任务是否存在。
func (s *Store) Update(id int, title *string, done *bool) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.items[id]
	if !ok {
		return Task{}, false
	}
	if title != nil {
		task.Title = *title
	}
	if done != nil {
		task.Done = *done
	}
	s.items[id] = task
	return task, true
}

// Delete 删除任务，返回是否真的删掉了。
func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}
