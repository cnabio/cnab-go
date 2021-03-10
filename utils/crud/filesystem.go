package crud

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ErrRecordDoesNotExist represents when file path is not found on file system
var ErrRecordDoesNotExist = errors.New("File does not exist")

// NewFileSystemStore creates a Store backed by a file system directory.
// Each key is represented by a file in that directory.
// - baseDirectory: the base directory under which files should be stored, e.g. /Users/carolynvs/.cnab
// - fileExtensions: map from item types (e.g. "claims") to the file extension that should be used (e.g. ".json")
func NewFileSystemStore(baseDirectory string, fileExtensions map[string]string) FileSystemStore {
	return FileSystemStore{
		baseDirectory:  baseDirectory,
		fileExtensions: fileExtensions,
	}
}

type FileSystemStore struct {
	baseDirectory string

	// Lookup of which file extension to use for which item type
	fileExtensions map[string]string
}

func (s FileSystemStore) Count(itemType string, group string) (int, error) {
	names, err := s.List(itemType, group)
	return len(names), err
}

func (s FileSystemStore) List(itemType string, group string) ([]string, error) {
	if err := s.ensure(itemType); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(filepath.Join(s.baseDirectory, itemType, group))
	if err != nil {
		// The group's directory doesn't exist, gracefully handle and continue
		if os.IsNotExist(err) {
			return []string{}, ErrRecordDoesNotExist
		}
		return []string{}, err
	}

	return s.names(itemType, s.storageFiles(itemType, files)), nil
}

func (s FileSystemStore) Save(itemType string, group string, name string, data []byte) error {
	filename, err := s.fullyQualifiedName(itemType, group, name)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, os.ModePerm)
}

func (s FileSystemStore) Read(itemType string, name string) ([]byte, error) {
	fileName, err := s.resolveFileName(itemType, name)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(fileName)
}

func (s FileSystemStore) Delete(itemType string, name string) error {
	filename, err := s.resolveFileName(itemType, name)
	if err != nil {
		return err
	}

	err = os.RemoveAll(filename)
	if err != nil {
		return err
	}

	return removeGroupDir(itemType, filename)
}

func (s FileSystemStore) resolveFileName(itemType string, name string) (string, error) {
	// First check for an exact match
	exactName, err := s.fullyQualifiedName(itemType, "", name)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(exactName); !os.IsNotExist(err) {
		return exactName, nil
	}

	// Fallback to looking in a subdirectory where we don't know the group's value
	wildName, err := s.fullyQualifiedName(itemType, "*", name)
	if err != nil {
		return "", err
	}

	matches, err := filepath.Glob(wildName)
	if err != nil {
		return "", err
	}

	switch len(matches) {
	case 0:
		return "", errors.Wrapf(ErrRecordDoesNotExist, "no file found for %s %s", itemType, name)
	case 1:
		fileName := matches[0]
		if _, err := os.Stat(fileName); err != nil {
			return "", errors.Wrapf(ErrRecordDoesNotExist, "cannot access %s", fileName)
		}
		return fileName, nil
	default:
		return "", fmt.Errorf("more than one file matched for %s %s", itemType, name)
	}
}

func removeGroupDir(itemType string, filename string) error {
	// Determine if parent directory represents a group dir
	if dir, _ := filepath.Split(filename); dir != itemType {
		// If so, read contents
		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer f.Close()

		// If empty, delete this group dir
		_, err = f.Readdir(1)
		if err == io.EOF {
			return os.RemoveAll(dir)
		}
	}
	return nil
}

func (s FileSystemStore) fileNameOf(itemType string, group string, name string) string {
	fileExt := s.fileExtensions[itemType]
	return filepath.Join(s.baseDirectory, itemType, group, fmt.Sprintf("%s%s", name, fileExt))
}

func (s FileSystemStore) fullyQualifiedName(itemType string, group string, name string) (string, error) {
	// Make sure the base path exists, ignoring wildcard groups
	var relPath string
	if group != "*" {
		relPath = filepath.Join(itemType, group)
	} else {
		relPath = itemType
	}
	if err := s.ensure(relPath); err != nil {
		return "", err
	}

	return s.fileNameOf(itemType, group, name), nil
}

func (s FileSystemStore) ensure(relPath string) error {
	target := filepath.Join(s.baseDirectory, relPath)
	fi, err := os.Stat(target)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("storage path %s exists, but is not a directory", target)
	}
	return os.MkdirAll(target, os.ModePerm)
}

func (s FileSystemStore) storageFiles(itemType string, files []os.FileInfo) []os.FileInfo {
	result := make([]os.FileInfo, 0)
	ext := s.fileExtensions[itemType]
	for _, file := range files {
		if file.IsDir() || ext == "" || filepath.Ext(file.Name()) == ext {
			result = append(result, file)
		}
	}
	return result
}

func (s FileSystemStore) names(itemType string, files []os.FileInfo) []string {
	result := make([]string, 0)
	for _, file := range files {
		result = append(result, s.name(itemType, file.Name()))
	}
	return result
}

func (s FileSystemStore) name(itemType string, path string) string {
	ext := s.fileExtensions[itemType]
	filename := filepath.Base(path)
	return strings.TrimSuffix(filename, ext)
}
