package util

import (
	"io/ioutil"
	"regexp"
)

func IOReadDir(root string) ([]string, error) {
	pattern := `/$`
	ispatternMatched, err := regexp.MatchString(pattern, root)
	if err != nil {
		return nil, err
	}
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, file := range fileInfo {
		if ispatternMatched {
			files = append(files, root+file.Name())
		} else {
			files = append(files, root+"/"+file.Name())
		}
	}
	return files, nil
}
