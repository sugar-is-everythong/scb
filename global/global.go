package global

import (
	"changeme/store"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	Local = "./local"
)

var (
	Store, _ = store.New(filepath.Join(Local, "store.gob"))
	App      *application.App
)
