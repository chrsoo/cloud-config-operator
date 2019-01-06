package v1alpha1

import (
	"io/ioutil"
	"os"
)

func testPath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func readFile(path string) string {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		// TODO log error
		panic(err)
	}
	return string(file)
}