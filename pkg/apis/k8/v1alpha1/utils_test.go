package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestTestPath(t *testing.T) {
	file := "banana.txt"
	exists := pathExists(file)
	assert.False(t, exists, "File '"+file+"' does not exist")

	tmp, err := ioutil.TempFile(os.TempDir(), "test-path-")
	defer os.Remove(tmp.Name())

	assert.Nil(t, err, "There should be no error")
	exists = pathExists(tmp.Name())
	assert.True(t, exists, "File '"+file+"' exists!")
}

func TestReadFile(t *testing.T) {
	tmp, err := ioutil.TempFile(os.TempDir(), "test-path-")
	assert.Nil(t, err, "Could not create temp file")
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString("Hello World!")
	assert.Nil(t, err, "Could write to temp file")
	tmp.Close()

	assert.Equal(t, "Hello World!", readFile(tmp.Name()))

}
