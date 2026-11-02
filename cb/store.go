//go:build windows

package cb

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store 全局实例，由 init() 初始化
var Store *StoreType

func init() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	Store, _ = NewStore(filepath.Join(dir, "local", "store.gob"))
}

// Entry 是一条历史记录，组合 Record 并加上时间戳。
type Entry struct {
	Record
	Time time.Time
}

// StoreType 把历史存在内存数组里，并在每次追加后整体写回文件。
type StoreType struct {
	mu      sync.Mutex
	entries []Entry
	path    string
}

// NewStore 创建存储并从 path 读回已有历史
func NewStore(path string) (*StoreType, error) {
	s := &StoreType{path: path}
	if path == "" {
		return s, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer f.Close()
	if err := gob.NewDecoder(f).Decode(&s.entries); err != nil && err != io.EOF {
		f.Close()
		os.Remove(path)
		return s, nil
	}
	return s, nil
}

// SetStoreR 以 Record 类型存储
func (s *StoreType) SetStoreR(r *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := Entry{Record: *r, Time: time.Now()}
	s.entries = append(s.entries, e)
	return s.flushLocked()
}

// SetStoreRJ 以 RecordJs 类型存储，先转成 Record
func (s *StoreType) SetStoreRJ(r *RecordJs) error {
	rec, err := r.transform()
	if err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	e := Entry{Record: *rec, Time: time.Now()}
	s.entries = append(s.entries, e)
	return s.flushLocked()
}

// All 返回全部历史（RecordJs 格式），新记录在前
func (s *StoreType) All() []RecordJs {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RecordJs, len(s.entries))
	for i, e := range s.entries {
		out[len(s.entries)-1-i] = e.Record.transform()
	}
	return out
}

// Delete 删除指定索引的条目（索引基于 All() 返回的顺序，新记录在前）
func (s *StoreType) Delete(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.entries) {
		return nil
	}
	// All() 返回倒序，index 0 对应 entries 最后一个
	realIndex := len(s.entries) - 1 - index
	s.entries = append(s.entries[:realIndex], s.entries[realIndex+1:]...)
	return s.flushLocked()
}

// SetAll 用 RecordJs 数组覆盖所有存储
func (s *StoreType) SetAll(records []RecordJs) error {
	entries := make([]Entry, 0, len(records))
	for _, r := range records {
		rec, err := r.transform()
		if err != nil {
			return err
		}
		entries = append(entries, Entry{Record: *rec, Time: time.Now()})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = entries
	return s.flushLocked()
}

// flushLocked 把当前数组整体写回 gob 文件
func (s *StoreType) flushLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := gob.NewEncoder(f).Encode(s.entries); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.path)
}
