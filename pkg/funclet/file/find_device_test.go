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

package file

import (
	"reflect"
	"testing"

	"github.com/baidu/openless/pkg/util/mount"
)

func TestManager_FindDevice(t *testing.T) {
	type fields struct {
		MountInfo []*mount.Info
	}
	type args struct {
		path string
	}
	normalMounts := []*mount.Info{
		&mount.Info{
			Mountpoint: "/tmp",
			Fstype:     "ext3",
			ID:         1,
		},
		&mount.Info{
			Mountpoint: "/var",
			Fstype:     "ext3",
			ID:         2,
		},
		&mount.Info{
			Mountpoint: "/var/run",
			Fstype:     "ext3",
			ID:         3,
		},
		&mount.Info{
			Mountpoint: "/home",
			Fstype:     "ext3",
			ID:         4,
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *mount.Info
		wantErr bool
	}{
		{
			name: "FindTmp",
			fields: fields{
				MountInfo: normalMounts,
			},
			args: args{
				path: "/tmp",
			},
			want: &mount.Info{
				Mountpoint: "/tmp",
				Fstype:     "ext3",
				ID:         1,
			},
		}, {
			name: "FindVar",
			fields: fields{
				MountInfo: normalMounts,
			},
			args: args{
				path: "/var/log",
			},
			want: &mount.Info{
				Mountpoint: "/var",
				Fstype:     "ext3",
				ID:         2,
			},
		}, {
			name: "FindExactlyMountPoint",
			fields: fields{
				MountInfo: normalMounts,
			},
			args: args{
				path: "/var/run",
			},
			want: &mount.Info{
				Mountpoint: "/var/run",
				Fstype:     "ext3",
				ID:         3,
			},
		}, {
			name: "FindVarRunSub",
			fields: fields{
				MountInfo: normalMounts,
			},
			args: args{
				path: "/var/run/a",
			},
			want: &mount.Info{
				Mountpoint: "/var/run",
				Fstype:     "ext3",
				ID:         3,
			},
		}, {
			name: "NotFound",
			fields: fields{
				MountInfo: normalMounts,
			},
			args: args{
				path: "/a",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MountManager{
				MountInfo: tt.fields.MountInfo,
			}
			got, err := d.FindDevice(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("MountManager.FindDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MountManager.FindDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
