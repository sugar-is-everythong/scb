package tools

import (
	"golang.org/x/sys/windows"
)

func GetPanName() []string {
	var drives []string
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	for i := 0; i < 26; i++ {
		if mask&(1<<i) != 0 {
			drives = append(drives, string(rune('A'+i)))
		}
	}
	return drives
}
