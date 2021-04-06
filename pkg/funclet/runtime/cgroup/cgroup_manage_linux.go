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

package cgroup

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/baidu/openless/pkg/api"

	runtimeApi "github.com/baidu/openless/pkg/funclet/runtime/api"

	"github.com/pkg/errors"

	"github.com/baidu/openless/pkg/util/bytefmt"

	libcontainercgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	libcontainerconfigs "github.com/opencontainers/runc/libcontainer/configs"

	"github.com/baidu/openless/pkg/util/logs"
)

// libcontainerCgroupManagerType defines how to interface with libcontainer
type libcontainerCgroupManagerType string

const (
	// libcontainerCgroupfs means use libcontainer with cgroupfs
	libcontainerCgroupfs libcontainerCgroupManagerType = "cgroupfs"
)

// libcontainerAdapter provides a simplified interface to libcontainer based on libcontainer type.
type libcontainerAdapter struct {
	// cgroupManagerType defines how to interface with libcontainer
	cgroupManagerType libcontainerCgroupManagerType
}

// newLibcontainerAdapter returns a configured libcontainerAdapter for specified manager.
// it does any initialization required by that manager to function.
func newLibcontainerAdapter(cgroupManagerType libcontainerCgroupManagerType) *libcontainerAdapter {
	return &libcontainerAdapter{cgroupManagerType: cgroupManagerType}
}

// newManager returns an implementation of cgroups.Manager
func (l *libcontainerAdapter) newManager(cgroups *libcontainerconfigs.Cgroup, paths map[string]string) (libcontainercgroups.Manager, error) {
	switch l.cgroupManagerType {
	case libcontainerCgroupfs:
		return &cgroupfs.Manager{
			Cgroups: cgroups,
			Paths:   paths,
		}, nil
	}
	return nil, fmt.Errorf("invalid cgroup manager configuration")
}

func (l *libcontainerAdapter) revertName(name string) CgroupName {
	return CgroupName(name)
}

// adaptName converts a CgroupName identifier to a driver specific conversion value.
// if outputToCgroupFs is true, the result is returned in the cgroupfs format rather than the driver specific form.
func (l *libcontainerAdapter) adaptName(cgroupName CgroupName, outputToCgroupFs bool) string {
	name := string(cgroupName)
	return name
}

// CgroupSubsystems holds information about the mounted cgroup subsytem
type CgroupSubsystems struct {
	// Cgroup subsystem mounts.
	// e.g.: "/sys/fs/cgroup/cpu" -> ["cpu", "cpuacct"]
	Mounts []libcontainercgroups.Mount

	// Cgroup subsystem to their mount location.
	// e.g.: "cpu" -> "/sys/fs/cgroup/cpu"
	MountPoints map[string]string
}

// cgroupManagerImpl implements the CgroupManager interface.
// Its a stateless object which can be used to
// update,create or delete any number of cgroups
// It uses the Libcontainer raw fs cgroup manager for cgroup management.
type cgroupManagerImpl struct {
	// subsystems holds information about all the
	// mounted cgroup subsystems on the node
	subsystems *CgroupSubsystems
	// cgroupRootPath
	cgroupRootPath string
	// simplifies interaction with libcontainer and its cgroup managers
	adapter *libcontainerAdapter
}

// Make sure that cgroupManagerImpl implements the CgroupManager interface
var _ CgroupManager = &cgroupManagerImpl{}

// NewCgroupManager is a factory method that returns a CgroupManager
func NewCgroupManager(cs *CgroupSubsystems, cgroupPath, cgroupDriver string) (m CgroupManager, err error) {
	path := DefaultCgroupPath
	if cgroupPath != "" {
		if filepath.Clean(cgroupPath) != cgroupPath || !filepath.IsAbs(cgroupPath) {
			return nil, errors.Errorf("invalid dir path %q", cgroupPath)
		}
		path = cgroupPath
	}

	managerType := libcontainerCgroupManagerType(cgroupDriver)
	return &cgroupManagerImpl{
		subsystems:     cs,
		cgroupRootPath: path,
		adapter:        newLibcontainerAdapter(managerType),
	}, nil
}

// Name converts the cgroup to the driver specific value in cgroupfs form.
func (m *cgroupManagerImpl) Name(name CgroupName) string {
	return m.adapter.adaptName(name, true)
}

// CgroupName converts the literal cgroupfs name on the host to an internal identifier.
func (m *cgroupManagerImpl) CgroupName(name string) CgroupName {
	return m.adapter.revertName(name)
}

// buildCgroupPaths builds a path to each cgroup subsystem for the specified name.
func (m *cgroupManagerImpl) buildCgroupPaths(name CgroupName) map[string]string {
	cgroupFsAdaptedName := m.Name(name)
	cgroupPaths := make(map[string]string, len(m.subsystems.MountPoints))
	for key, val := range m.subsystems.MountPoints {
		cgroupPaths[key] = path.Join(val, cgroupFsAdaptedName)
	}
	return cgroupPaths
}

// Exists checks if all subsystem cgroups already exist
func (m *cgroupManagerImpl) Exists(name CgroupName) bool {
	// Get map of all cgroup paths on the system for the particular cgroup
	cgroupPaths := m.buildCgroupPaths(name)

	// the presence of alternative control groups not known to runc confuses
	// the kubelet existence checks.
	// ideally, we would have a mechanism in runc to support Exists() logic
	// scoped to the set control groups it understands.  this is being discussed
	// in https://github.com/opencontainers/runc/issues/1440
	// once resolved, we can remove this code.
	whitelistControllers := NewString("cpu", "cpuacct", "cpuset", "memory", "systemd")

	// If even one cgroup path doesn't exist, then the cgroup doesn't exist.
	for controller, path := range cgroupPaths {
		// ignore mounts we don't care about
		if !whitelistControllers.Has(controller) {
			continue
		}
		if !libcontainercgroups.PathExists(path) {
			return false
		}
	}

	return true
}

// Destroy destroys the specified cgroup
func (m *cgroupManagerImpl) Destroy(cgroupConfig *CgroupConfig) error {

	cgroupPaths := m.buildCgroupPaths(cgroupConfig.Name)

	// we take the location in traditional cgroupfs format.
	abstractCgroupFsName := string(cgroupConfig.Name)
	abstractParent := CgroupName(path.Dir(abstractCgroupFsName))
	abstractName := CgroupName(path.Base(abstractCgroupFsName))

	driverParent := m.adapter.adaptName(abstractParent, false)
	driverName := m.adapter.adaptName(abstractName, false)

	// Initialize libcontainer's cgroup config with driver specific naming.
	libcontainerCgroupConfig := &libcontainerconfigs.Cgroup{
		Name:   driverName,
		Parent: driverParent,
	}

	manager, err := m.adapter.newManager(libcontainerCgroupConfig, cgroupPaths)
	if err != nil {
		return err
	}

	// Delete cgroups using libcontainers Managers Destroy() method
	if err = manager.Destroy(); err != nil {
		return fmt.Errorf("Unable to destroy cgroup paths for cgroup %v : %v", cgroupConfig.Name, err)
	}

	return nil
}

// GetResourcesConfig takes the readable input resource params and outputs the cgroup resource config
func (m *cgroupManagerImpl) GetResourcesConfig(r *ResourceParams) (config *runtimeApi.ResourceConfig, err error) {
	mem, err := bytefmt.ToBytes(r.MemLimits)
	if err != nil {
		return nil, err
	}

	var memoryLimits, cpuRequests, cpuLimits int64
	var cpuQuota int64
	var cpuShares, cpuPeriod uint64
	memoryLimits = int64(mem)

	if r.CPURequests == -1 {
		cpuShares = SharesPerCPU
	} else {
		cpuRequests = int64(r.CPURequests * 1000)
		cpuShares = MilliCPUToShares(cpuRequests)
	}

	cgroupCpuPath, err := m.GetCgroupSubsysPath("cpu")
	if err != nil {
		return nil, err
	}
	cpuPeriod, err = GetCgroupParamUint(cgroupCpuPath, CFSPeriodFile)
	if err != nil {
		return nil, err
	}

	if r.CPULimits == -1 {
		cpuQuota = int64(-1)
	} else {
		cpuLimits = int64(r.CPULimits * 1000)
		cpuQuota = MilliCPUToQuota(cpuLimits, int64(cpuPeriod))

	}
	logs.Debugf("[GetResourcesConfig] cpu request %d, cpu shares %d, cpu period %d, cpu quota %d", cpuRequests, cpuShares, cpuPeriod, cpuQuota)
	config = &runtimeApi.ResourceConfig{
		Memory:    &memoryLimits,
		CpuShares: &cpuShares,
		CpuPeriod: &cpuPeriod,
		CpuQuota:  &cpuQuota,
	}
	return
}

func (m *cgroupManagerImpl) GetCgroupSubsysPath(subsys string) (subsysPath string, err error) {
	for _, sys := range supportedSubsystems {
		if sys.Name() == subsys {
			return path.Join(m.cgroupRootPath, subsys), nil
		}
	}
	return "", ErrUnsupportedCgroup{CgroupName: subsys}
}

type subsystem interface {
	// Name returns the name of the subsystem.
	Name() string
	// Set the cgroup represented by cgroup.
	Set(path string, cgroup *libcontainerconfigs.Cgroup) error
	// GetStats returns the statistics associated with the cgroup
	GetStats(path string, stats *libcontainercgroups.Stats) error
}

// Cgroup subsystems we currently support
var supportedSubsystems = []subsystem{
	&cgroupfs.MemoryGroup{},
	&cgroupfs.CpuGroup{},
	&cgroupfs.CpuacctGroup{},
	&cgroupfs.FreezerGroup{},
}

// setSupportedSubsytems sets cgroup resource limits only on the supported
// subsytems. ie. cpu and memory. We don't use libcontainer's cgroup/fs/Set()
// method as it doesn't allow us to skip updates on the devices cgroup
// Allowing or denying all devices by writing 'a' to devices.allow or devices.deny is
// not possible once the device cgroups has children. Once the pod level cgroup are
// created under the QOS level cgroup we cannot update the QOS level device cgroup.
// We would like to skip setting any values on the device cgroup in this case
// but this is not possible with libcontainers Set() method
// See https://github.com/opencontainers/runc/issues/932
func setSupportedSubsystems(cgroupConfig *libcontainerconfigs.Cgroup) error {
	for _, sys := range supportedSubsystems {
		if _, ok := cgroupConfig.Paths[sys.Name()]; !ok {
			return fmt.Errorf("Failed to find subsytem mount for subsytem: %v", sys.Name())
		}
		if err := sys.Set(cgroupConfig.Paths[sys.Name()], cgroupConfig); err != nil {
			return fmt.Errorf("Failed to set config for supported subsystems : %v", err)
		}
	}
	return nil
}

func (m *cgroupManagerImpl) toResources(resourceConfig *runtimeApi.ResourceConfig) *libcontainerconfigs.Resources {
	resources := &libcontainerconfigs.Resources{}
	if resourceConfig == nil {
		return resources
	}

	if resourceConfig.Memory != nil {
		resources.Memory = *resourceConfig.Memory
	}
	if resourceConfig.MemorySwap != nil {
		resources.MemorySwap = *resourceConfig.MemorySwap
	}
	if resourceConfig.CpuShares != nil {
		resources.CpuShares = *resourceConfig.CpuShares
	}
	if resourceConfig.CpuQuota != nil {
		resources.CpuQuota = *resourceConfig.CpuQuota
	}
	if resourceConfig.CpuPeriod != nil {
		resources.CpuPeriod = *resourceConfig.CpuPeriod
	}
	if resourceConfig.Freezer != nil {
		resources.Freezer = libcontainerconfigs.FreezerState(*resourceConfig.Freezer)
	} else {
		resources.Freezer = libcontainerconfigs.Undefined
	}
	return resources
}

// Update updates the cgroup with the specified Cgroup Configuration
func (m *cgroupManagerImpl) Update(cgroupConfig *CgroupConfig) error {

	// Extract the cgroup resource parameters
	resourceConfig := cgroupConfig.ResourceParameters
	resources := m.toResources(resourceConfig)

	cgroupPaths := m.buildCgroupPaths(cgroupConfig.Name)

	// we take the location in traditional cgroupfs format.
	abstractCgroupFsName := string(cgroupConfig.Name)
	abstractParent := CgroupName(path.Dir(abstractCgroupFsName))
	abstractName := CgroupName(path.Base(abstractCgroupFsName))

	driverParent := m.adapter.adaptName(abstractParent, false)
	driverName := m.adapter.adaptName(abstractName, false)

	// Initialize libcontainer's cgroup config
	libcontainerCgroupConfig := &libcontainerconfigs.Cgroup{
		Name:      driverName,
		Parent:    driverParent,
		Resources: resources,
		Paths:     cgroupPaths,
	}

	if err := setSupportedSubsystems(libcontainerCgroupConfig); err != nil {
		return fmt.Errorf("failed to set supported cgroup subsystems for cgroup %v: %v", cgroupConfig.Name, err)
	}
	return nil
}

// Create creates the specified cgroup
func (m *cgroupManagerImpl) Create(cgroupConfig *CgroupConfig) error {

	// we take the location in traditional cgroupfs format.
	abstractCgroupFsName := string(cgroupConfig.Name)
	abstractParent := CgroupName(path.Dir(abstractCgroupFsName))
	abstractName := CgroupName(path.Base(abstractCgroupFsName))

	driverParent := m.adapter.adaptName(abstractParent, false)
	driverName := m.adapter.adaptName(abstractName, false)

	resources := m.toResources(cgroupConfig.ResourceParameters)
	// Initialize libcontainer's cgroup config with driver specific naming.
	libcontainerCgroupConfig := &libcontainerconfigs.Cgroup{
		Name:      driverName,
		Parent:    driverParent,
		Resources: resources,
	}

	// get the manager with the specified cgroup configuration
	manager, err := m.adapter.newManager(libcontainerCgroupConfig, nil)
	if err != nil {
		return err
	}

	// Apply(-1) is a hack to create the cgroup directories for each resource
	// subsystem. The function [cgroups.Manager.apply()] applies cgroup
	// configuration to the process with the specified pid.
	// It creates cgroup files for each subsystems and writes the pid
	// in the tasks file. We use the function to create all the required
	// cgroup files but not attach any "real" pid to the cgroup.
	if err := manager.Apply(-1); err != nil {
		return err
	}

	// it may confuse why we call set after we do apply, but the issue is that runc
	// follows a similar pattern.  it's needed to ensure cpu quota is set properly.
	m.Update(cgroupConfig)

	return nil
}

// Scans through all subsystems to find pids associated with specified cgroup.
func (m *cgroupManagerImpl) Pids(name CgroupName) []int {
	// we need the driver specific name
	cgroupFsName := m.Name(name)

	// Get a list of processes that we need to kill
	pidsToKill := NewInt()
	var pids []int
	for _, val := range m.subsystems.MountPoints {
		dir := path.Join(val, cgroupFsName)
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			// The subsystem pod cgroup is already deleted
			// do nothing, continue
			continue
		}
		// Get a list of pids that are still charged to the pod's cgroup
		pids, err = getCgroupProcs(dir)
		if err != nil {
			continue
		}
		pidsToKill.Insert(pids...)

		// WalkFunc which is called for each file and directory in the pod cgroup dir
		visitor := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logs.V(4).Infof("cgroup manager encountered error scanning cgroup path %q: %v", path, err)
				return filepath.SkipDir
			}
			if !info.IsDir() {
				return nil
			}
			pids, err = getCgroupProcs(path)
			if err != nil {
				logs.V(4).Infof("cgroup manager encountered error getting procs for cgroup path %q: %v", path, err)
				return filepath.SkipDir
			}
			pidsToKill.Insert(pids...)
			return nil
		}
		// Walk through the pod cgroup directory to check if
		// container cgroups haven't been GCed yet. Get attached processes to
		// all such unwanted containers under the pod cgroup
		if err = filepath.Walk(dir, visitor); err != nil {
			logs.V(4).Infof("cgroup manager encountered error scanning pids for directory: %q: %v", dir, err)
		}
	}
	return pidsToKill.List()
}

// ReduceCPULimits reduces the cgroup's cpu shares to the lowest possible value
func (m *cgroupManagerImpl) ReduceCPULimits(cgroupName CgroupName) error {
	// Set lowest possible CpuShares value for the cgroup
	minimumCPUShares := uint64(MinShares)
	resources := &runtimeApi.ResourceConfig{
		CpuShares: &minimumCPUShares,
	}
	containerConfig := &CgroupConfig{
		Name:               cgroupName,
		ResourceParameters: resources,
	}
	return m.Update(containerConfig)
}

func getStatsSupportedSubsystems(cgroupPaths map[string]string) (*libcontainercgroups.Stats, error) {
	stats := libcontainercgroups.NewStats()
	for _, sys := range supportedSubsystems {
		if _, ok := cgroupPaths[sys.Name()]; !ok {
			return nil, fmt.Errorf("Failed to find subsystem mount for subsystem: %v", sys.Name())
		}
		if err := sys.GetStats(cgroupPaths[sys.Name()], stats); err != nil {
			return nil, fmt.Errorf("Failed to get stats for supported subsystems : %v", err)
		}
	}
	return stats, nil
}

func toResourceStats(stats *libcontainercgroups.Stats) *api.ResourceStats {
	return &api.ResourceStats{
		MemoryStats: &api.MemoryStats{
			Usage:     int64(stats.MemoryStats.Usage.Usage),
			Limit:     int64(stats.MemoryStats.Usage.Limit),
			SwapLimit: int64(stats.MemoryStats.SwapUsage.Limit),
		},
		CPUStats: &api.CPUStats{
			TotalUsage: int64(stats.CpuStats.CpuUsage.TotalUsage),
		},
	}
}

// Get sets the ResourceParameters of the specified cgroup as read from the cgroup fs
func (m *cgroupManagerImpl) GetResourceStats(name CgroupName) (*api.ResourceStats, error) {
	cgroupPaths := m.buildCgroupPaths(name)
	stats, err := getStatsSupportedSubsystems(cgroupPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats supported cgroup subsystems for cgroup %v: %v", name, err)
	}
	freezerState, err := GetFreezerStats(cgroupPaths["freezer"])
	if err != nil {
		return nil, err
	}
	resourceStats := toResourceStats(stats)
	resourceStats.FreezerState = freezerState
	return resourceStats, nil
}

func (m *cgroupManagerImpl) GetResourceConfig(name CgroupName) (resourceConfig *runtimeApi.ResourceConfig, err error) {
	cgroupPaths := m.buildCgroupPaths(name)
	var CPUPath, memoryPath, freezerPath string
	for controller, p := range cgroupPaths {
		switch controller {
		case "cpu":
			CPUPath = p
		case "memory":
			memoryPath = p
		case "freezer":
			freezerPath = p
		default:
		}
	}

	memoryLimit, err := GetMemoryLimit(memoryPath)
	if err != nil {
		return nil, err
	}
	cpuPeriod, err := GetCPUPeriod(CPUPath)
	if err != nil {
		return nil, err
	}
	cpuShares, err := GetCPUShares(CPUPath)
	if err != nil {
		return nil, err
	}
	cpuQuota, err := GetCPUQuota(CPUPath)
	if err != nil {
		return nil, err
	}
	freeze, err := GetFreezerStats(freezerPath)
	if err != nil {
		return nil, err
	}
	resourceConfig = &runtimeApi.ResourceConfig{
		Memory:    &memoryLimit,
		CpuPeriod: &cpuPeriod,
		CpuQuota:  &cpuQuota,
		CpuShares: &cpuShares,
		Freezer:   &freeze,
	}
	return resourceConfig, nil
}
