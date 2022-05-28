package executor

import "sync"

type Manager struct {
	mu   sync.RWMutex
	data map[string]*Job
}

// SetJob 设置数据
func (t *Manager) SetJob(key string, job *Job) {
	t.mu.Lock()
	t.data[key] = job
	t.mu.Unlock()
}

// GetJob 获取数据
func (t *Manager) GetJob(key string) *Job {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data[key]
}

// GetJobs 获取数据
func (t *Manager) GetJobs() map[string]*Job {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data
}

// DelJob 设置数据
func (t *Manager) DelJob(key string) {
	t.mu.Lock()
	delete(t.data, key)
	t.mu.Unlock()
}

// Len 长度
func (t *Manager) Len() int {
	return len(t.data)
}

// Exists Key是否存在
func (t *Manager) Exists(key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.data[key]
	return ok
}
