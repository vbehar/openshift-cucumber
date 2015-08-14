/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nfs

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/types"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util/mount"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/volume"

	"github.com/golang/glog"
)

// This is the primary entrypoint for volume plugins.
// The recyclerConfig arg provides the ability to configure recycler behavior.  It is implemented as a pointer to allow nils.
// The nfsPlugin is used to store the recyclerConfig and give it, when needed, to the func that creates NFS Recyclers.
// Tests that exercise recycling should not use this func but instead use ProbeRecyclablePlugins() to override default behavior.
func ProbeVolumePlugins(recyclerConfig *volume.RecyclableVolumeConfig) []volume.VolumePlugin {
	return []volume.VolumePlugin{
		&nfsPlugin{
			host:            nil,
			newRecyclerFunc: newRecycler,
			recyclerConfig:  recyclerConfig,
		},
	}
}

type nfsPlugin struct {
	host volume.VolumeHost
	// decouple creating recyclers by deferring to a function.  Allows for easier testing.
	newRecyclerFunc func(spec *volume.Spec, host volume.VolumeHost, recyclerConfig *volume.RecyclableVolumeConfig) (volume.Recycler, error)
	recyclerConfig  *volume.RecyclableVolumeConfig
}

var _ volume.VolumePlugin = &nfsPlugin{}
var _ volume.PersistentVolumePlugin = &nfsPlugin{}
var _ volume.RecyclableVolumePlugin = &nfsPlugin{}

const (
	nfsPluginName = "kubernetes.io/nfs"
)

func (plugin *nfsPlugin) Init(host volume.VolumeHost) {
	plugin.host = host
}

func (plugin *nfsPlugin) Name() string {
	return nfsPluginName
}

func (plugin *nfsPlugin) CanSupport(spec *volume.Spec) bool {
	return spec.VolumeSource.NFS != nil || spec.PersistentVolumeSource.NFS != nil
}

func (plugin *nfsPlugin) GetAccessModes() []api.PersistentVolumeAccessMode {
	return []api.PersistentVolumeAccessMode{
		api.ReadWriteOnce,
		api.ReadOnlyMany,
		api.ReadWriteMany,
	}
}

func (plugin *nfsPlugin) NewBuilder(spec *volume.Spec, pod *api.Pod, _ volume.VolumeOptions, mounter mount.Interface) (volume.Builder, error) {
	return plugin.newBuilderInternal(spec, pod, mounter)
}

func (plugin *nfsPlugin) newBuilderInternal(spec *volume.Spec, pod *api.Pod, mounter mount.Interface) (volume.Builder, error) {
	var source *api.NFSVolumeSource

	if spec.VolumeSource.NFS != nil {
		source = spec.VolumeSource.NFS
	} else {
		source = spec.PersistentVolumeSource.NFS
	}
	return &nfs{
		volName:    spec.Name,
		server:     source.Server,
		exportPath: source.Path,
		readOnly:   source.ReadOnly,
		mounter:    mounter,
		pod:        pod,
		plugin:     plugin,
	}, nil

}

func (plugin *nfsPlugin) NewCleaner(volName string, podUID types.UID, mounter mount.Interface) (volume.Cleaner, error) {
	return plugin.newCleanerInternal(volName, podUID, mounter)
}

func (plugin *nfsPlugin) newCleanerInternal(volName string, podUID types.UID, mounter mount.Interface) (volume.Cleaner, error) {
	return &nfs{
		volName:    volName,
		server:     "",
		exportPath: "",
		readOnly:   false,
		mounter:    mounter,
		pod:        &api.Pod{ObjectMeta: api.ObjectMeta{UID: podUID}},
		plugin:     plugin,
	}, nil
}

func (plugin *nfsPlugin) NewRecycler(spec *volume.Spec) (volume.Recycler, error) {
	if plugin.recyclerConfig == nil {
		return nil, fmt.Errorf("RecyclableVolumeConfig is nil for this plugin.  Recycler cannot be created.")
	}
	return plugin.newRecyclerFunc(spec, plugin.host, plugin.recyclerConfig)
}

func newRecycler(spec *volume.Spec, host volume.VolumeHost, recyclableConfig *volume.RecyclableVolumeConfig) (volume.Recycler, error) {
	if spec.VolumeSource.HostPath != nil {
		return &nfsRecycler{
			name:             spec.Name,
			server:           spec.VolumeSource.NFS.Server,
			path:             spec.VolumeSource.NFS.Path,
			host:             host,
			recyclableConfig: recyclableConfig,
		}, nil
	} else {
		return &nfsRecycler{
			name:             spec.Name,
			server:           spec.PersistentVolumeSource.NFS.Server,
			path:             spec.PersistentVolumeSource.NFS.Path,
			host:             host,
			recyclableConfig: recyclableConfig,
		}, nil
	}
}

// NFS volumes represent a bare host file or directory mount of an NFS export.
type nfs struct {
	volName    string
	pod        *api.Pod
	server     string
	exportPath string
	readOnly   bool
	mounter    mount.Interface
	plugin     *nfsPlugin
	// decouple creating recyclers by deferring to a function.  Allows for easier testing.
	newRecyclerFunc func(spec *volume.Spec, host volume.VolumeHost) (volume.Recycler, error)
}

// SetUp attaches the disk and bind mounts to the volume path.
func (nfsVolume *nfs) SetUp() error {
	return nfsVolume.SetUpAt(nfsVolume.GetPath())
}

func (nfsVolume *nfs) SetUpAt(dir string) error {
	mountpoint, err := nfsVolume.mounter.IsMountPoint(dir)
	glog.V(4).Infof("NFS mount set up: %s %v %v", dir, mountpoint, err)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if mountpoint {
		return nil
	}
	os.MkdirAll(dir, 0750)
	source := fmt.Sprintf("%s:%s", nfsVolume.server, nfsVolume.exportPath)
	options := []string{}
	if nfsVolume.readOnly {
		options = append(options, "ro")
	}
	err = nfsVolume.mounter.Mount(source, dir, "nfs", options)
	if err != nil {
		mountpoint, mntErr := nfsVolume.mounter.IsMountPoint(dir)
		if mntErr != nil {
			glog.Errorf("IsMountpoint check failed: %v", mntErr)
			return err
		}
		if mountpoint {
			if mntErr = nfsVolume.mounter.Unmount(dir); mntErr != nil {
				glog.Errorf("Failed to unmount: %v", mntErr)
				return err
			}
			mountpoint, mntErr := nfsVolume.mounter.IsMountPoint(dir)
			if mntErr != nil {
				glog.Errorf("IsMountpoint check failed: %v", mntErr)
				return err
			}
			if mountpoint {
				// This is very odd, we don't expect it.  We'll try again next sync loop.
				glog.Errorf("%s is still mounted, despite call to unmount().  Will try again next sync loop.", dir)
				return err
			}
		}
		os.Remove(dir)
		return err
	}
	return nil
}

func (nfsVolume *nfs) GetPath() string {
	name := nfsPluginName
	return nfsVolume.plugin.host.GetPodVolumeDir(nfsVolume.pod.UID, util.EscapeQualifiedNameForDisk(name), nfsVolume.volName)
}

func (nfsVolume *nfs) TearDown() error {
	return nfsVolume.TearDownAt(nfsVolume.GetPath())
}

func (nfsVolume *nfs) TearDownAt(dir string) error {
	mountpoint, err := nfsVolume.mounter.IsMountPoint(dir)
	if err != nil {
		glog.Errorf("Error checking IsMountPoint: %v", err)
		return err
	}
	if !mountpoint {
		return os.Remove(dir)
	}

	if err := nfsVolume.mounter.Unmount(dir); err != nil {
		glog.Errorf("Unmounting failed: %v", err)
		return err
	}
	mountpoint, mntErr := nfsVolume.mounter.IsMountPoint(dir)
	if mntErr != nil {
		glog.Errorf("IsMountpoint check failed: %v", mntErr)
		return mntErr
	}
	if !mountpoint {
		if err := os.Remove(dir); err != nil {
			return err
		}
	}

	return nil
}

// nfsRecycler scrubs an NFS volume by running "rm -rf" on the volume in a pod.
type nfsRecycler struct {
	name             string
	server           string
	path             string
	host             volume.VolumeHost
	recyclableConfig *volume.RecyclableVolumeConfig
}

func (r *nfsRecycler) GetPath() string {
	return r.path
}

// Recycler provides methods to reclaim the volume resource.
// A NFS volume is recycled by scheduling a pod to run "rm -rf" on the contents of the volume.
// Recycle blocks until the pod has completed or any error occurs.
// The scrubber pod's is expected to succeed within 5 minutes else an error will be returned.
func (r *nfsRecycler) Recycle() error {
	// TODO:  remove the duplication between this Recycle func and the one in host_path.go
	pod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			GenerateName: "pv-scrubber-" + util.ShortenString(r.name, 44) + "-",
			Namespace:    api.NamespaceDefault,
		},
		Spec: api.PodSpec{
			ActiveDeadlineSeconds: &r.recyclableConfig.Timeout,
			RestartPolicy:         api.RestartPolicyNever,
			Volumes: []api.Volume{
				{
					Name: "vol",
					VolumeSource: api.VolumeSource{
						NFS: &api.NFSVolumeSource{
							Server: r.server,
							Path:   r.path,
						},
					},
				},
			},
			Containers: []api.Container{
				{
					Name:    "scrubber",
					Image:   r.recyclableConfig.ImageName,
					Command: r.recyclableConfig.Command,
					Args:    r.recyclableConfig.Args,
					VolumeMounts: []api.VolumeMount{
						{
							Name:      "vol",
							MountPath: "/scrub",
						},
					},
				},
			},
		},
	}
	return volume.ScrubPodVolumeAndWatchUntilCompletion(pod, r.host.GetKubeClient())
}
