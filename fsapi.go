package main

import (
	//"bazil.org/fuse"
	//"bazil.org/fuse/fs"
	//"bazil.org/fuse/fuseutil"
	"bytes"
	"errors"
	//"fmt"
	"github.com/Jeffail/gabs"
	"github.com/Sirupsen/logrus"
	//"github.com/patrickmn/go-cache"
	//"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	//"strconv"
	//"strings"
	//"sync"
	//"syscall"
	//"time"
)

func (f *File) ReadFile() (string, error) {
	cp_consoles_url := site + "/api/consoles/" + f.dir.fs.container + "/read_file?path=" + f.path

	req, err := http.NewRequest("GET", cp_consoles_url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.dir.fs.token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("read_file %v", err)
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return "", errors.New(ERROR_NOT_AUTHORIZED)
	}
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

//Need to change this to Dir.ListFiles
func ListFiles(access_token string, container_name string, path string) ([]File, error) {
	//cache_key := container_name + ":" + path
	var FileCollection []File
	/*FileCollectionCache, found := cp_cache.Get(cache_key)
	  if found {
	      FileCollection = FileCollectionCache.([]File)
	  } else {*/

	cp_consoles_url := site + "/api/consoles/" + container_name + "/files?path=" + path
	req, err := http.NewRequest("GET", cp_consoles_url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+access_token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("List files %v", err)
		panic(err)
	}
	defer resp.Body.Close()
	/*
	   if resp.StatusCode == 401 {
	       return FileCollection, errors.New(ERROR_NOT_AUTHORIZED)
	   }*/

	body, err := ioutil.ReadAll(resp.Body)
	jsonFiles, err := gabs.ParseJSON(body)
	jsonPaths, _ := jsonFiles.ChildrenMap()
	for key, child := range jsonPaths {
		var jsonFile File
		jsonFile.name = string(key)

		jsonFile.path = child.Path("path").Data().(string)
		jsonFile.mime = child.Path("type").Data().(string)
		jsonFile.size = uint64(child.Path("size").Data().(float64))
		FileCollection = append(FileCollection, jsonFile)

	}
	//cp_cache.Set(cache_key, FileCollection, cache.DefaultExpiration)
	//}
	return FileCollection, nil
}

func (d *Dir) CreateDir(newdir string) (err error) {
	cp_consoles_url := site + "/api/consoles/" + d.fs.container + "/create_folder"
	cp_payload := ` { "path": "` + newdir + `" }`
	var jsonStr = []byte(cp_payload)

	req, err := http.NewRequest("POST", cp_consoles_url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.fs.token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	if err != nil {
		logrus.Errorf("CreateDir %v", err)
		return err
	}
	/*cache_key := d.fs.container + ":" + d.path
	cp_cache.Delete(cache_key)*/
	return nil
}

func IsOffline(file string) bool {
	var is_offline bool
	//Users may see what appear to be random, zero-byte files appear in their home directory, named 4913, 5036, 5159, 5282 (increasing at increments of 123.)

	offline_regexp := []string{`^.+?\.sw.+$`, `^.+?~$`, `^4913$`, `^\._.+?$`}
	for _, reg := range offline_regexp {
		is_offline, _ = regexp.MatchString(reg, file)
		if is_offline == true {
			return true
		}
	}
	return false
}

//func (d *Dir) TouchFile(file string, ch chan error) (err error) {
func (d *Dir) TouchFile(file string) (err error) {
	cp_consoles_url := site + "/api/consoles/" + d.fs.container + "/exec"
	var cp_payload string
	cp_payload = ` { "commands": "touch ` + file + `" }`
	var jsonStr = []byte(cp_payload)

	req, err := http.NewRequest("POST", cp_consoles_url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.fs.token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		//ch <- errors.New(ERROR_NOT_AUTHORIZED)
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	if err != nil {
		logrus.Errorf("CreateFile %v", err)
		//ch <- err
		return err
	}
	//ch <- err
	return nil
}

func (f *File) UploadFile() (err error) {
	cp_consoles_url := site + "/api/consoles/" + f.dir.fs.container + "/upload_file"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	temp_file, err := ioutil.TempFile(os.TempDir(), "cp_")
	err = ioutil.WriteFile(temp_file.Name(), f.data, 0666)
	if err != nil {
		logrus.Errorf("Writint temp %v", err)
		return err
	}
	fw, err := w.CreateFormFile("file", temp_file.Name())
	if err != nil {
		logrus.Errorf("CreateFormFile %v", err)
		return err
	}
	if _, err = io.Copy(fw, temp_file); err != nil {
		return
	}
	if fw, err = w.CreateFormField("path"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("/app/" + f.dir.path + "/" + f.name)); err != nil {
		return
	}
	w.Close()
	req, err := http.NewRequest("POST", cp_consoles_url, &b)
	if err != nil {
		logrus.Errorf("Upload Request %v \n", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+f.dir.fs.token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", user_agent)

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode == 401 {
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	if err != nil {
		logrus.Errorf("Remove temp_file %v", err)
	}
	/*cache_key := f.dir.fs.container + ":" + f.dir.path
	cp_cache.Delete(cache_key)*/
	return
}

//need to merge RemoveDir and RemoveFile
func (d *Dir) RemoveFile(file string) (err error) {
	cp_consoles_url := site + "/api/consoles/" + d.fs.container + "/exec"
	var cp_payload string
	if d.path == "" {
		cp_payload = ` { "commands": "rm ` + file + `" }`
	} else {
		cp_payload = ` { "commands": "rm ` + d.path + "/" + file + `" }`
	}
	var jsonStr = []byte(cp_payload)

	req, err := http.NewRequest("POST", cp_consoles_url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.fs.token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("RemoveFile %v", err)
		return err
	}
	if resp.StatusCode == 401 {
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	//logrus.Infof("Remove file End %s", d.path+" / "+file)
	return nil
}

func (d *Dir) RemoveDir(dir string) (err error) {
	cp_consoles_url := site + "/api/consoles/" + d.fs.container + "/exec"
	var cp_payload string
	//logrus.Infof("Remove file %s", d.path+" / "+file)
	if dir == "" {
		//Avoid remove base directory
		//logrus.Infof("RemoveDir empty dir %s", dir)
		//cp_payload = ` { "commands": "rm ` + dir + `" }`
		return nil
	} else if d.path == "" {
		//logrus.Infof("RemoveDir empty d.path %s", d.path)
		cp_payload = ` { "commands": "rm -rf /app/` + dir + `" }`
	} else {
		cp_payload = ` { "commands": "rm -rf /app/` + d.path + "/" + dir + `" }`
	}
	//logrus.Infof("RemoveDir payload %s", cp_payload)
	var jsonStr = []byte(cp_payload)

	req, err := http.NewRequest("POST", cp_consoles_url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.fs.token)
	req.Header.Set("User-Agent", user_agent)
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("RemoveDir %v", err)
		return err
	}
	if resp.StatusCode == 401 {
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	//logrus.Infof("Remove dir End %s", d.path+" / "+dir)
	return nil
}

func (f *File) UploadAsyncFile(ch chan error) (err error) {
	cp_consoles_url := site + "/api/consoles/" + f.dir.fs.container + "/upload_file"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	temp_file, err := ioutil.TempFile(os.TempDir(), "cp_")
	err = ioutil.WriteFile(temp_file.Name(), f.data, 0666)
	if err != nil {
		logrus.Errorf("Writint temp %v", err)
		ch <- err
		return err
	}
	fw, err := w.CreateFormFile("file", temp_file.Name())
	if err != nil {
		logrus.Errorf("CreateFormFile %v", err)
		ch <- err
		return err
	}
	if _, err = io.Copy(fw, temp_file); err != nil {
		return
	}
	if fw, err = w.CreateFormField("path"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("/app/" + f.dir.path + "/" + f.name)); err != nil {
		return
	}
	w.Close()
	req, err := http.NewRequest("POST", cp_consoles_url, &b)
	if err != nil {
		logrus.Errorf("Upload Request %v \n", err)
		ch <- err
		return err
	}
	req.Header.Set("Authorization", "Bearer "+f.dir.fs.token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", user_agent)

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		ch <- err
		return err
	}
	if resp.StatusCode == 401 {
		ch <- errors.New(ERROR_NOT_AUTHORIZED)
		return errors.New(ERROR_NOT_AUTHORIZED)
	}
	if err != nil {
		logrus.Errorf("Remove temp_file %v", err)
	}
	/*cache_key := f.dir.fs.container + ":" + f.dir.path
	cp_cache.Delete(cache_key)*/
	ch <- err
	return
}