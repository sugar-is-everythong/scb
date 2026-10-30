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
			"HasRTF:", rec.HasRTF,
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
	ctx := context.Background()
	c, err := Watch(ctx, 500*time.Millisecond)
	if err != nil {
		fmt.Println(err.Error())
	}
	for {
		select {
		case r := <-c:
			fmt.Println(r)
		}
	}

	time.Sleep(1 * time.Hour)
}
