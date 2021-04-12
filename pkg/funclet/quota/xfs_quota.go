/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package device
package quota

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/baidu/easyfaas/pkg/funclet/storage"

	"github.com/baidu/easyfaas/pkg/util/file"

	funletCmd "github.com/baidu/easyfaas/pkg/funclet/command"

	"github.com/baidu/easyfaas/pkg/util/logs"
)

var (
	// projectFile: the file where we store mappings between project ids and directories
	// http://man7.org/linux/man-pages/man5/projects.5.html
	// its content format is like this:
	// 10:/export/cage
	// 42:/var/log
	projectsFile = "/etc/projects"

	// // projid: the file where we store mappings between project ids and human readable names
	// // http://man7.org/linux/man-pages/man5/projid.5.html
	// // its content format is like this:
	// // cage:10
	// // logfiles:42
	// projectIdFile = "/etc/projid"

	idPoolSize = 1000
)

type xfsQuotaCtrl struct {
	// projects: mapping projectIds and directories
	projects       *map[uint32]string
	pathIDs        *map[string]uint32
	projIDPool     chan uint32
	RunnersTmpPath string
	tmpSize        string
	logger         *logs.Logger
	dataLock       sync.Mutex
	fileLock       sync.Mutex
}

func InitQuotaCtrl(storagePath string, tmpSize string, runnersTmpPath string, storageType string, logger *logs.Logger) (quotactrl *xfsQuotaCtrl, err error) {
	quotactrl = &xfsQuotaCtrl{
		tmpSize:  tmpSize,
		dataLock: sync.Mutex{},
		fileLock: sync.Mutex{},
		logger:   logger,
	}

	st, err := storage.GetStorage(storageType)
	if err != nil {
		quotactrl.logger.Errorf("init tmp storage type %s failed: %s", storageType, err)
		return nil, err
	}

	if err := st.PrepareXfs(storagePath); err != nil {
		quotactrl.logger.Errorf("storage %s prepare xfs failed: %s", storageType, err)
		return nil, err
	}

	// prepare runner root tmp directory
	quotactrl.logger.Infof("[init xfs quota] start mkdir runner tmp")
	if err := os.MkdirAll(runnersTmpPath, 0777); err != nil {
		quotactrl.logger.Errorf("[init xfs quota] mkdir runner tmp failed: %s", err)
		return nil, err
	}
	quotactrl.RunnersTmpPath = runnersTmpPath

	if err := st.MountProjQuota(storagePath, runnersTmpPath); err != nil {
		quotactrl.logger.Errorf("storage %s mount runner tmp path %s failed: %s", storagePath, runnersTmpPath, err)
		return nil, err
	}

	// prepare project files
	quotactrl.logger.Infof("[init xfs quota] start prepare project file")
	if err := quotactrl.prepareProjectFiles(); err != nil {
		quotactrl.logger.Errorf("[init xfs quota] prepare project file failed: %s", err)
		return nil, err
	}

	// parse all project info
	quotactrl.logger.Infof("[init xfs quota] start parse exist projects")
	if err := quotactrl.syncExistsProjects(); err != nil {
		quotactrl.logger.Errorf("[init xfs quota] parse exist projects failed: %s", err)
		return nil, err
	}

	// init project ids pool
	quotactrl.logger.Infof("[init xfs quota] start init project ids pool")
	quotactrl.initProjectIDsPool()

	return
}

func (q *xfsQuotaCtrl) AllocTmpDevice(path string) (err error) {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0777); err != nil {
				q.logger.Errorf("[alloc tmp device] mkdir tmp path %s failed: %s", path, err)
				return err
			}
		} else {
			return err
		}
	}
	if f.Mode() != os.ModePerm {
		if err := os.Chmod(path, os.ModePerm); err != nil {
			q.logger.Errorf("[alloc tmp device] chmod tmp path %s failed: %s", path, err)
			return err
		}
	}
	if err := q.addProjectInfo(path); err != nil {
		q.logger.Errorf("[alloc tmp device] add project info path %s failed: %s", path, err)
		return err
	}
	defer func() {
		if err != nil {
			if err := q.removeProjectInfo(path); err != nil {
				q.logger.Errorf("[alloc tmp device] reset project info path %s failed: %s", path, err)
			}
		}
	}()
	if err := q.setQuota(path); err != nil {
		q.logger.Errorf("[alloc tmp device] set project quota path %s failed: %s", path, err)
		return err
	}
	return nil
}

func (q *xfsQuotaCtrl) FreeTmpDevice(path string) error {
	if err := q.removeProjectInfo(path); err != nil {
		q.logger.Errorf("[free tmp device] remove project info path %s failed: %s", path, err)
		return err
	}

	if err := file.EraseFile(path); err != nil {
		q.logger.Errorf("[free tmp device] erase tmp file data path %s failed: %s", path, err)
		return err
	}
	return nil
}

func (q *xfsQuotaCtrl) prepareProjectFiles() error {
	if _, err := os.Stat(projectsFile); os.IsNotExist(err) {
		file, cerr := os.Create(projectsFile)
		if cerr != nil {
			return fmt.Errorf("error creating xfs projects file %s: %v", projectsFile, cerr)
		}
		file.Close()
	}
	return nil
}

func (q *xfsQuotaCtrl) syncExistsProjects() error {
	RunWithLog(q.fileLock.Lock, "syncExistsProjects file lock")
	projectMap := map[uint32]string{}
	pathMap := map[string]uint32{}
	f, err := os.OpenFile(projectsFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		RunWithLog(q.fileLock.Unlock, "syncExistsProjects file unlock")
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ol := sc.Text()
		items := strings.Split(ol, ":")
		idInt, err := strconv.Atoi(items[0])
		if err != nil {
			logs.Warnf("project line %s parse err", ol)
			// TODO: When parse error occurred, check the content. make a decision whether to remove it
			continue
		}
		id := uint32(idInt)
		projectMap[id] = items[1]
		pathMap[items[1]] = id
	}
	if err := sc.Err(); err != nil {
		RunWithLog(q.fileLock.Unlock, "syncExistsProjects file unlock")
		return err
	}
	RunWithLog(q.fileLock.Unlock, "syncExistsProjects file unlock")

	RunWithLog(q.dataLock.Lock, "syncExistsProjects data lock")
	defer RunWithLog(q.dataLock.Unlock, "syncExistsProjects data unlock")
	q.projects = &projectMap
	q.pathIDs = &pathMap
	return nil
}

func (q *xfsQuotaCtrl) initProjectIDsPool() {
	q.projIDPool = make(chan uint32, idPoolSize)
	var projID uint32 = 1
	size := 1
	for {
		if size > idPoolSize {
			break
		}
		if _, ok := (*q.projects)[projID]; ok {
			projID++
			continue
		}
		q.projIDPool <- projID
		size++
		projID++
	}
	return
}

func (q *xfsQuotaCtrl) addProjectInfo(path string) error {
	nextID := q.getNextProjectID()
	projectDesc := fmt.Sprintf("%d:%s\n", nextID, path)
	if err := q.addToFile(projectDesc); err != nil {
		return err
	}

	iDStr := strconv.FormatUint(uint64(nextID), 10)
	out, err := funletCmd.CommandOutput("xfs_quota", "-x", "-c", fmt.Sprintf("project -s %s", iDStr), q.RunnersTmpPath)
	if err != nil {
		logs.Errorf("xfs quota set project out: [%s] \n err: [%s]", out, err)
		return fmt.Errorf("xfs_quota failed with error: %v, output: %s", err, out)
	}
	logs.Infof("xfs quota set project out: [%s] \n err: [%s]", out, err)
	RunWithLog(q.dataLock.Lock, "addProjectInfo data lock")
	(*q.projects)[nextID] = path
	(*q.pathIDs)[path] = nextID
	RunWithLog(q.dataLock.Unlock, "addProjectInfo data unlock")
	return nil
}

func (q *xfsQuotaCtrl) removeProjectInfo(path string) error {
	if err := q.removeFromFile(path); err != nil {
		return err
	}
	RunWithLog(q.dataLock.Lock, "removeProjectInfo  data lock")
	projID := (*q.pathIDs)[path]
	delete((*q.projects), projID)
	delete((*q.pathIDs), path)
	RunWithLog(q.dataLock.Unlock, "removeProjectInfo data unlock")
	q.projIDPool <- projID
	return nil
}

func (q *xfsQuotaCtrl) getProjectID(path string) (id uint32, ok bool) {
	RunWithLog(q.dataLock.Lock, "getProjectID data lock")
	defer RunWithLog(q.dataLock.Unlock, "getProjectID data unlock")
	id, ok = (*q.pathIDs)[path]
	return
}

func (q *xfsQuotaCtrl) getNextProjectID() uint32 {
	id := <-q.projIDPool
	return id
}

func (q *xfsQuotaCtrl) addToFile(toAdd string) error {
	RunWithLog(q.fileLock.Lock, "addToFile file lock")
	defer RunWithLog(q.fileLock.Unlock, "addToFile file unlock")

	file, err := os.OpenFile(projectsFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.WriteString(toAdd); err != nil {
		return err
	}
	file.Sync()

	return nil
}

func (q *xfsQuotaCtrl) removeFromFile(path string) error {
	RunWithLog(q.fileLock.Lock, "removeFromFile file lock")
	defer RunWithLog(q.fileLock.Unlock, "removeFromFile file unlock")
	input, err := ioutil.ReadFile(projectsFile)
	if err != nil {
		return err
	}
	regStr := fmt.Sprintf("(?m)[\r\n]*^.*%s$", path)
	q.logger.Infof("remove from file reg string is %s", regStr)
	re := regexp.MustCompile(regStr)
	res := re.ReplaceAllString(string(input), "")
	if err := ioutil.WriteFile(projectsFile, []byte(res), 0644); err != nil {
		return err
	}
	return nil
}

func (q *xfsQuotaCtrl) setQuota(path string) error {
	id, ok := q.getProjectID(path)
	if !ok {
		return fmt.Errorf("the path %s not belongs to any projects", path)
	}
	iDStr := strconv.FormatUint(uint64(id), 10)

	out, err := funletCmd.CommandOutput("xfs_quota", "-x", "-c", fmt.Sprintf("limit -p bhard=%s %s", q.tmpSize, iDStr), q.RunnersTmpPath)
	if err != nil {
		q.logger.Errorf("xfs quota set quota out: [%s] \n err: [%s]", out, err)
		return fmt.Errorf("xfs_quota failed with error: %v, output: %s", err, out)
	}
	q.logger.Infof("xfs quota set quota out: [%s] \n err: [%s]", out, err)
	return nil
}

func (q *xfsQuotaCtrl) SnapshotProjectPaths() (paths *[]string, err error) {
	if err := q.syncExistsProjects(); err != nil {
		q.logger.Errorf("[snapshot projects] sync exist projects failed: %s", err)
		return nil, err
	}
	RunWithLog(q.dataLock.Lock, "snapshot projects data lock")
	defer RunWithLog(q.dataLock.Unlock, "snapshot projects data unlock")
	pathArr := make([]string, 0)
	for _, path := range *q.projects {
		pathArr = append(pathArr, path)
	}
	return &pathArr, nil
}

func RunWithLog(fn func(), des string) {
	logs.V(9).Infof("start do %s", des)
	fn()
	logs.V(9).Infof("finish do %s", des)
}
