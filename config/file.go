package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils/fileutils"
	"github.com/pkg/errors"
)

func resolveConfigFilePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Search for the relative path to the file in the config folder, taking into account
	// various common starting points.
	if configFile := fileutils.FindFile(filepath.Join("config", path)); configFile != "" {
		return configFile, nil
	}

	// Search for the relative path in the current working directory, also taking into account
	// various common starting points.
	if configFile := fileutils.FindPath(path, []string{"."}, nil); configFile != "" {
		return configFile, nil
	}

	// Otherwise, search for the config/ folder using the same heuristics as above, and build
	// an absolute path anchored there and joining the given input path (or plain filename).
	if configFolder, found := fileutils.FindDir("config"); found {
		return filepath.Join(configFolder, path), nil
	}

	// Fail altogether if we can't even find the config/ folder. This should only happen if
	// the executable is relocated away from the supporting files.
	return "", fmt.Errorf("failed to find config file %s", path)
}

type FileStore struct {
	commonStore

	path string
}

func NewFileStore(path string) (fs *FileStore, err error) {
	resolvedPath, err := resolveConfigFilePath(path)
	if err != nil {
		return nil, err
	}

	fs = &FileStore{
		path: resolvedPath,
	}
	if err = fs.Load(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileStore) persist(cfg *model.Config) error {
	b, err := marshalConfig(cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fs.path, b, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileStore) Load() (err error) {
	var needsSave bool
	var f io.ReadCloser

	f, err = os.Open(fs.path)
	if os.IsNotExist(err) {
		needsSave = true
		defaultCfg := &model.Config{}
		defaultCfg.SetDefaults()

		var defaultCfgBytes []byte
		defaultCfgBytes, err = marshalConfig(defaultCfg)
		if err != nil {
			return err
		}

		f = ioutil.NopCloser(bytes.NewReader(defaultCfgBytes))

	} else if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if err == nil && closeErr != nil {
			err = errors.Wrap(closeErr, "failed to close")
		}
	}()

	return fs.commonStore.load(f, needsSave, fs.commonStore.validate, fs.persist)
}
