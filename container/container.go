package container

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	RUNNING = iota
	PAUSE
	STOPPED

	MAX_CONTAINER_COUNT = 1024
)

var (
	AvailableContainerIds = [MAX_CONTAINER_COUNT]int8{0}
)

type Container struct {
	Id                  string
	RootPath            string
	Status              int8
	LogPath             string
	ElfPatcherPath      string
	FakechrootPath      string
	SettingConfPath     string
	SettingConf         map[string]interface{}
	StartTime           string
	ImageName           string
	ContainerName       string
	CreateUser          string
	MemcachedServerList string
	ShmFiles            string
	IpcFiles            string
}

func findAvailableId() (int, *Error) {
	for i := 0; i < MAX_CONTAINER_COUNT; i++ {
		if AvailableContainerIds[i] == 0 {
			AvailableContainerIds[i] = 1
			return i, nil
		} else {
			continue
		}
	}
	cerr := ErrNew(ErrFull, "No available container id could be generated")
	return -1, &cerr
}

func createContainer(dir string, name string) (*Container, *Error) {
	id, err := findAvailableId()
	if err == nil {
		var con Container
		con.Id = fmt.Sprintf("container-%d", id)
		for strings.HasSuffix(dir, "/") {
			dir = strings.TrimSuffix(dir, "/")
		}
		con.RootPath = fmt.Sprintf("%s/%s/instance", dir, con.Id)
		con.Status = STOPPED
		con.LogPath = fmt.Sprintf("%s/%s/log", dir, con.Id)
		con.ElfPatcherPath = fmt.Sprintf("%s/%s/elf/", dir, con.Id)
		con.FakechrootPath = fmt.Sprintf("%s/%s/fakechroot/", dir, con.Id)
		con.SettingConfPath = fmt.Sprintf("%s/%s/settings/", dir, con.Id)
		con.SettingConf, _ = GetMap("setting.yml", []string{con.SettingConfPath})
		con.ImageName = name
		return &con, nil
	}
	return nil, err
}

func RunContainer(dir string, name string) (*Container, *Error) {
	fmt.Printf("dir:%s, name: %s\n", dir, name)
	return nil, nil
}

func DestroyContainer(name string) (*Container, *Error) {
	fmt.Printf("name: %s\n", name)
	return nil, nil
}

func Walkfs(con *Container) ([]string, *Error) {
	fileList := []string{}

	err := filepath.Walk(con.RootPath, func(path string, f os.FileInfo, err error) error {
		ftype, err := FileType(path)
		if err != nil {
			return err
		}
		if ftype == TYPE_REGULAR {
			_, err := FilePermission(path, PERM_EXE)
			if err != nil {
				return err
			}
			fileList = append(fileList, path)
		}
		return nil
	})
	cerr := ErrNew(err, "walkfs error")
	return fileList, &cerr
}

func Command(cmdStr string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		cerr := ErrNew(err, "cmd running error")
		return "", &cerr
	}
	return out.String(), nil
}

func CommandEnv(cmdStr string, env map[string]string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	for key, value := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		cerr := ErrNew(err, "commandenv error")
		return "", &cerr
	}
	return out.String(), nil
}
