package utils

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"kusionstack.io/kpm/pkg/errors"
	"kusionstack.io/kpm/pkg/reporter"
)

// HashDir computes the checksum of a directory by concatenating all files and
// hashing them by sha256.
func HashDir(dir string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// files in the ".git "directory will cause the same repository, cloned at different times,
		// has different checksum.
		if strings.Contains(path, ".git") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}

// StoreToFile will store 'data' into toml file under 'filePath'.
func StoreToFile(filePath string, dataStr string) error {
	file, err := os.Create(filePath)
	if err != nil {
		reporter.ExitWithReport("kpm: failed to create file: ", filePath, err)
		return err
	}
	defer file.Close()

	file, err = os.Create(filePath)

	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.WriteString(file, dataStr); err != nil {
		return err
	}
	return nil
}

// ParseRepoNameFromGitUrl get the repo name from git url,
// the repo name in 'https://github.com/xxx/kcl1.git' is 'kcl1'.
func ParseRepoNameFromGitUrl(gitUrl string) string {
	name := filepath.Base(gitUrl)
	return name[:len(name)-len(filepath.Ext(name))]
}

// CreateFileIfNotExist will check whether the file under a certain path 'filePath/fileName' exists,
// and return an error if it exists, and call the method 'storeFunc' to save the file if it does not exist.
func CreateFileIfNotExist(filePath string, storeFunc func() error) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		reporter.Report("kpm: creating new :", filePath)
		err := storeFunc()
		if err != nil {
			reporter.Report("kpm: failed to create: ,", filePath)
			return err
		}
	} else {
		reporter.Report("kpm: '" + filePath + "' already exists")
		return err
	}
	return nil
}

// Whether the file exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// todo: Consider using the OCI tarball as the standard tar format.
var ignores = []string{".git", ".tar"}

func TarDir(srcDir string, tarPath string) error {

	fw, err := os.Create(tarPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fw.Close()

	tw := tar.NewWriter(fw)
	defer tw.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, ignore := range ignores {
			if strings.Contains(path, ignore) {
				return nil
			}
		}

		relPath, _ := filepath.Rel(srcDir, path)
		relPath = filepath.ToSlash(relPath)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = relPath

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		fr, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fr.Close()

		if _, err := io.Copy(tw, fr); err != nil {
			return err
		}

		return nil
	})

	return err
}

func DirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DefaultKpmHome create the '.kpm' in the user home and return the path of ".kpm".
func CreateDefaultKpmHome() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.InternalBug
	}

	dirPath := filepath.Join(homeDir, ".kpm")
	if !DirExists(dirPath) {
		err = os.Mkdir(dirPath, 0755)
		if err != nil {
			return "", errors.InternalBug
		}
	}

	return dirPath, nil
}

// GetAbsKpmHome will return the absolute path of $KPM_HOME,
// or the absolute path of the current path if $KPM_HOME does not exist.
func GetAbsKpmHome() (string, error) {
	kpmHome := os.Getenv("KPM_HOME")
	if kpmHome == "" {
		defaultHome, err := CreateDefaultKpmHome()
		if err != nil {
			return "", errors.InternalBug
		}
		kpmHome = defaultHome
	}

	kpmHome, err := filepath.Abs(kpmHome)
	if err != nil {
		return "", errors.InternalBug
	}

	return kpmHome, nil
}

// CreateSymlink will create symbolic link named 'newName' for 'oldName',
// and if the symbolic link already exists, it will be deleted and recreated.
func CreateSymlink(oldName, newName string) error {
	if DirExists(newName) {
		err := os.Remove(oldName)
		if err != nil {
			return errors.InternalBug
		}
	}

	err := os.Symlink(oldName, newName)
	if err != nil {
		return err
	}
	return nil
}