package cb

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCb(t *testing.T) {
	/*	rec, err := Read()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("HasText:", rec.HasText,
			"HasImage:", rec.HasImage,
			"HasHTML:", rec.HasHTML,
			"HasFiles:", rec.HasFiles)

		if rec.HasText {
			fmt.Println("text:", rec.GetText())
		}
		if rec.HasImage {
			pngData, err := rec.GetImage()
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("image PNG bytes:", len(pngData))
		}
		if rec.HasHTML {
			h, err := rec.GetHTML()
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("html source:", h.SourceURL)
			fmt.Println("html fragment:", h.Fragment)
		}
		if rec.HasFiles {
			files, err := rec.GetFiles()
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("files:", files)
		}

		WriteText("哈哈哈")*/
	time.Sleep(1 * time.Second)
	recordChan, _ := Watch(context.Background(), 500*time.Millisecond)
	for rec := range recordChan {
		f, _ := rec.GetFiles()
		h, _ := rec.GetHTML()
		fmt.Printf("HasFiles:%t,%v\n", rec.HasFiles, f)
		fmt.Printf("HasText:%t\n", rec.HasText)
		fmt.Printf("HasHTML:%t,%s\n", rec.HasHTML, h.Fragment)
		fmt.Printf("HasImage:%t\n\n", rec.HasImage)
	}

	time.Sleep(1 * time.Hour)
}
