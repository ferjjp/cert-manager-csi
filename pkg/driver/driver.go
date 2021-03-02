/*
Copyright 2019 The Jetstack cert-manager contributors.

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

package driver

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/jetstack/cert-manager-csi/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	Version = "0.1.1"
)

type Driver struct {
	endpoint string

	ids *identityServer
	cs  *ControllerServer
	ns  *NodeServer
}

func New(driverName, nodeID, endpoint, dataRoot, tmpfsSize string) (*Driver, error) {
	glog.Infof("driver: %v version: %v", driverName, Version)

	mntPoint, err := util.IsLikelyMountPoint(dataRoot)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dataRoot, 0700); err != nil {
			return nil, status.Error(codes.Internal,
				fmt.Sprintf("failed to create data root directory %s: %s", dataRoot, err))
		}

		mntPoint = false
	}

	if !mntPoint {
		execErr := new(bytes.Buffer)
		cmd := exec.Command("mount", "-F", "tmpfs", "-o", "size="+tmpfsSize+"m", "swap", dataRoot)
		cmd.Stderr = execErr

		if err := cmd.Run(); err != nil {
			glog.Errorf("node: failed to mount data root (%s): %s",
				execErr.String(), err)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	ns, err := NewNodeServer(nodeID, dataRoot, tmpfsSize)
	if err != nil {
		return nil, err
	}

	return &Driver{
		endpoint: endpoint,
		ids:      NewIdentityServer(driverName, Version),
		cs:       NewControllerServer(),
		ns:       ns,
	}, nil
}

func (d *Driver) Run() {
	s := NewNonBlockingGRPCServer()
	s.Start(d.endpoint, d.ids, d.cs, d.ns)
	s.Wait()
}

func (d *Driver) NodeServer() *NodeServer {
	return d.ns
}
