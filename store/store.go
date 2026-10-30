// Package store 提供剪贴板历史的持久化：内存数组 + gob 二进制文件
// 图片等 []byte 以原始二进制直接存进文件。
package store

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry 是一条历史记录，内容由调用方决定填哪些字段（零值字段即"没有"）。
type Entry struct {
	Text  string
	Image []byte // PNG 编码；原始字节直接存进文件
	HTML  string
	RTF   string
	Files []string
	Time  time.Time
}

// Store 把历史存在内存数组里，并在每次追加后整体写回文件。
type Store struct {
	mu      sync.Mutex
	entries []Entry
	path    string // gob 文件路径；空表示只存内存不落盘
}

// New 创建存储并从 path 读回已有历史；文件不存在、为空或损坏均视为空历史。
// 损坏时删掉坏文件重新开始（历史数据非关键，无恢复价值）；返回错误仅剩权限等系统级 IO 问题。
func New(path string) (*Store, error) {
	s := &Store{path: path}
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
	// 空文件解码返回 io.EOF，视为空历史
	if err := gob.NewDecoder(f).Decode(&s.entries); err != nil && err != io.EOF {
		// 文件损坏：关闭句柄后删除坏文件（Windows 不允许删除打开中的文件），下次 Save 重建
		f.Close()
		os.Remove(path)
		return s, nil
	}
	return s, nil
}

// Save 追加一条记录并立即落盘；Time 未填时自动取当前时间。
func (s *Store) Save(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	s.entries = append(s.entries, e)
	return s.flushLocked()
}

// All 返回全部历史，新记录在前。
func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	for i, e := range s.entries {
		out[len(s.entries)-1-i] = e
	}
	return out
}

// flushLocked 把当前数组整体写回 gob 文件（调用方需持锁）。
// 先写临时文件再改名，避免写一半崩溃留下坏文件。
func (s *Store) flushLocked() error {
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
