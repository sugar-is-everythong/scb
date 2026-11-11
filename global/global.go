package global

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Local 是数据目录的绝对路径，锚定可执行文件所在目录。
// 不能用相对路径：进程工作目录会漂移，历史索引和图片文件的落点必须恒定；
// 取不到可执行文件路径时直接在包初始化阶段 panic，而不是把故障拖到运行期。
var Local = func() string {
	exe, err := os.Executable()
	if err != nil {
		panic("global: 无法获取可执行文件路径: " + err.Error())
	}
	return filepath.Join(filepath.Dir(exe), "local")
}()

var (
	App        *application.App
	Foreground uintptr // 工具窗口显示前的句柄
)
