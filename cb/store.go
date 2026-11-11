//go:build windows

package cb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"scb/global"
	"sync"
)

// Store 全局实例，由 init() 初始化
var Store *StoreType

func init() {
	var err error
	Store, err = NewStore(filepath.Join(global.Local, "store"))
	if err != nil {
		panic("cb: 历史索引初始化失败: " + err.Error())
	}
}

// StoreType 把历史存在内存数组里（旧→新），每次变更后整体序列化为 JSON 落盘。
// 二进制成员不再进索引：图片以记录 id 为文件名单独落在数据目录，
// 记录里只存 cb.URL 形式的可直接访问地址，落盘体量只有纯文本。
type StoreType struct {
	mu      sync.Mutex
	entries []RecordJs
	path    string // 索引文件路径，空表示不落盘
}

// NewStore 创建存储并从 path 读回已有历史。
// path 是"无后缀文本文件"约定：内容就是标准 JSON 数组，只是不挂 .json 后缀。
func NewStore(path string) (*StoreType, error) {
	s := &StoreType{path: path}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.entries); err != nil {
			// 解析失败保留文件供人工排查（文本格式的意义所在），只从空历史继续；
			// 不能删文件，否则用户手改出一个语法错误就会把全部历史连带清空。
			fmt.Println("cb: 历史索引解析失败，从空历史继续:", err)
			s.entries = nil
		}
	}
	return s, nil
}

// SetStoreRJ 追加一条记录：先把内存图片与外部文件持久化到数据目录并改写地址，
// 再入数组整体落盘。存入的就是 RecordJs 本身，不再经 Record 中转。
// 顺序不能反：persistImage 先拷 BUrl 对应的图并改写 files 匹配项，
// persistFiles 再处理剩余外部文件，已入自家目录的会自动跳过，同一文件不会拷两遍。
func (s *StoreType) SetStoreRJ(r *RecordJs) error {
	rec := *r
	if err := persistImage(&rec); err != nil {
		// 落盘中途失败：记录不会入库，已写的图片/文件副本必须清掉，否则成孤儿文件
		removeLocalImageFiles(rec.Id)
		return err
	}
	if err := persistFiles(&rec); err != nil {
		removeLocalImageFiles(rec.Id)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, rec)
	return s.flushLocked()
}

// All 返回全部历史（RecordJs 格式），新记录在前
func (s *StoreType) All() []RecordJs {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RecordJs, len(s.entries))
	for i, e := range s.entries {
		out[len(s.entries)-1-i] = e
	}
	return out
}

// Delete 删除指定索引的条目（索引基于 All() 返回的顺序，新记录在前），
// 并清掉本程序写在数据目录里的图片文件。
func (s *StoreType) Delete(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.entries) {
		return nil
	}
	// All() 返回倒序，index 0 对应 entries 最后一个
	realIndex := len(s.entries) - 1 - index
	e := s.entries[realIndex]
	s.entries = append(s.entries[:realIndex], s.entries[realIndex+1:]...)
	removeLocalImageFiles(e.Id)
	return s.flushLocked()
}

// SetAll 用 RecordJs 数组覆盖所有存储（入参顺序为旧→新，与 All() 相反）
func (s *StoreType) SetAll(records []RecordJs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = records
	return s.flushLocked()
}

// flushLocked 把当前数组整体写回索引文件。
// 图片字节已拆出单独落盘，索引只剩纯文本，全量重写的体量恒定且小。
func (s *StoreType) flushLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(s.entries)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.path)
}
