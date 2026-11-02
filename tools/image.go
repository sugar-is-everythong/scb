package tools

import (
	"io"
	"net/http"
)

func GetWebImage(url string) ([]byte, error) {

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	imgData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return imgData, nil
}
