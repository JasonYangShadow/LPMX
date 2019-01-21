package docker

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/docker/distribution/manifest"
	. "github.com/docker/distribution/manifest/schema1"
	"github.com/docker/libtrust"
	. "github.com/jasonyangshadow/lpmx/error"
	registry "github.com/jasonyangshadow/lpmx/registry"
	. "github.com/jasonyangshadow/lpmx/utils"
	digest "github.com/opencontainers/go-digest"
)

const (
	DOCKER_URL  = "https://registry-1.docker.io"
	SETTING_URL = "https://raw.githubusercontent.com/JasonYangShadow/LPMXSettingRepository/master"
)

func ListRepositories(username string, pass string) ([]string, *Error) {
	log.SetOutput(ioutil.Discard)
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, cerr
	}
	repo, err := hub.Repositories()
	if err != nil {
		cerr := ErrNew(err, "query docker repositories failure")
		return nil, cerr
	}
	return repo, nil
}

func ListTags(username string, pass string, name string) ([]string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, cerr
	}
	tags, err := hub.Tags(name)
	if err != nil {
		cerr := ErrNew(err, "query docker tags failure")
		return nil, cerr
	}
	return tags, nil
}

func GetDigest(username string, pass string, name string, tag string) (string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return "", cerr
	}
	digest, err := hub.ManifestDigest(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker digest failure")
		return "", cerr
	}
	return digest.String(), nil
}

func UploadManifests(username string, pass string, name string, tag string) *Error {
	log.SetOutput(ioutil.Discard)

	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return cerr
	}

	man := &Manifest{
		Versioned: Versioned{
			SchemaVersion: 1,
		},
		Tag: tag,
	}
	key, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		cerr := ErrNew(err, "libtrust generates private key error")
		return cerr
	}

	signedManifest, err := Sign(man, key)
	if err != nil {
		cerr := ErrNew(err, "signing manifest error")
		return cerr
	}

	err = hub.PutManifest(name, tag, signedManifest)
	if err != nil {
		cerr := ErrNew(err, "putting manifest error")
		return cerr
	}
	return nil
}

func UploadLayers(username string, pass string, name string, tag string, file string) (string, *Error) {
	log.SetOutput(ioutil.Discard)

	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return "", cerr
	}

	shasum, cerr := Sha256file(file)
	if cerr != nil {
		return "", cerr
	}

	dig := digest.NewDigestFromHex(
		"sha256",
		shasum,
	)
	exists, err := hub.HasBlob(name, dig)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("qury dig from repo %s error", name))
		return shasum, cerr
	}

	if !exists {
		data, err := os.Open(file)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not open %s for reading", file))
			return shasum, cerr
		}
		defer data.Close()
		herr := hub.UploadBlob(name, dig, data)
		if herr != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not upload %s", file))
			return shasum, cerr
		}
	}
	return shasum, nil
}

func DeleteManifest(username string, pass string, name string, tag string) *Error {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return cerr
	}
	digest, err := hub.ManifestDigest(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker digest failure")
		return cerr
	}
	err = hub.DeleteManifest(name, digest)
	if err != nil {
		cerr := ErrNew(err, "delete docker manifest failure")
		return cerr
	}
	return nil
}

func DownloadLayers(username string, pass string, name string, tag string, folder string) (map[string]int64, []string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
		name = "library/" + name
	}
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return nil, nil, err
		}
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, nil, cerr
	}
	man, err := hub.ManifestV2(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker manifest failure")
		return nil, nil, cerr
	}
	data := make(map[string]int64)
	var layer_order []string
	for _, element := range man.Layers {
		dig := element.Digest
		//reader, err := hub.DownloadLayer(name, dig)
		//function name is changed
		reader, err := hub.DownloadBlob(name, dig)
		if err != nil {
			cerr := ErrNew(err, "download docker layers failure")
			return nil, nil, cerr
		}
		defer reader.Close()
		if strings.HasSuffix(folder, "/") {
			folder = strings.TrimSuffix(folder, "/")
		}
		filename := folder + "/" + strings.TrimPrefix(dig.String(), "sha256:")
		to, err := os.Create(filename)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("create file %s failure", filename))
			return nil, nil, cerr
		}
		defer to.Close()
		fmt.Println(fmt.Sprintf("Downloading file with type: %s, size: %d", element.MediaType, element.Size))

		//printing download percentage using anonymous functions
		go func(filename string, size int64) {
			f, err := os.Open(filename)
			if err != nil {
				return
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				return
			}
			curr_size := fi.Size()
			for curr_size < size {
				percentage := int(float64(curr_size) / float64(size) * 100)
				fmt.Printf("Downloading... %d/%d [%d/100 complete]", curr_size, size, percentage)
				time.Sleep(time.Second)
				fi, err = f.Stat()
				curr_size = fi.Size()
				fmt.Printf("\r")
			}
		}(filename, element.Size)

		if _, err := io.Copy(to, reader); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("copy file %s content failure", filename))
			return nil, nil, cerr
		}
		data[filename] = element.Size
		layer_order = append(layer_order, filename)
	}
	return data, layer_order, nil
}

func DownloadSetting(name string, tag string, folder string) *Error {
	filepath := fmt.Sprintf("%s/setting.yml", folder)
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return err
		}
	}
	out, err := os.Create(filepath)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("%s file create error", filepath))
		return cerr
	}
	defer out.Close()

	name = strings.ToLower(name)
	tag = strings.ToLower(tag)

	http_req := fmt.Sprintf("%s/%s/%s/setting.yml", SETTING_URL, name, tag)
	resp, err := http.Get(http_req)
	if err != nil {
		cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
		return cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		http_req := fmt.Sprintf("%s/default.yml", SETTING_URL)
		resp, err := http.Get(http_req)
		if err != nil {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
			return cerr
		}
		return nil
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
		return cerr
	}
	return nil
}

func DownloadFilefromGithub(name string, tag string, filename string, url string, folder string) *Error {
	filepath := fmt.Sprintf("%s/%s", folder, filename)
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return err
		}
	}

	out, err := os.Create(filepath)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("%s file create error", filepath))
		return cerr
	}
	defer out.Close()

	name = strings.ToLower(name)
	tag = strings.ToLower(tag)
	http_req := fmt.Sprintf("%s/%s/%s/%s", url, name, tag, filename)
	resp, err := http.Get(http_req)
	if err != nil {
		cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
		return cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		http_req := fmt.Sprintf("%s/default.%s", url, filename)
		resp, err := http.Get(http_req)
		if err != nil {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
			return cerr
		}
		return nil
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
		return cerr
	}
	return nil

}
