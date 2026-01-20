// Copyright 2026 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	"github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (f *CloudProviderFramework) GetLustreSummaryByDisplayName(ctx context.Context, compartmentId, adLocation, pvName string) (*lustrefilestorage.LustreFileSystemSummary, error) {
	Logf("GetLustreFileSystemSummaryByDisplayName request params")
	Logf("compartmentId: %+v", compartmentId)
	Logf("adLocation: %+v", adLocation)
	Logf("pvName: %+v", pvName)
	fsVolumeSummaryList, err := f.Client.Lustre(nil).ListLustreFileSystems(ctx, compartmentId, adLocation, pvName)
	if client.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if len(fsVolumeSummaryList) == 0 {
		Logf("fsVolumeSummaryList is empty or nil")
		return nil, fmt.Errorf("no Lustre file system volume found")
	}

	Logf("fsVolumeSummaryList length: %d", len(fsVolumeSummaryList))
	Logf("First volume summary: %+v", fsVolumeSummaryList[0])

	return &fsVolumeSummaryList[0], nil
}

func (f *CloudProviderFramework) GetLustreFSIdByDisplayName(ctx context.Context, compartmentId, adLocation, pvName string) (string, error) {
	fsSummary, err := f.GetLustreSummaryByDisplayName(ctx, compartmentId, adLocation, pvName)
	if err != nil {
		return "", err
	}
	return *fsSummary.Id, nil
}

func (f *CloudProviderFramework) CheckLustreVolumeExist(ctx context.Context, fsId string) bool {
	fs, err := f.Client.Lustre(nil).GetLustreFileSystem(ctx, fsId)
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		return false
	}
	if fs.LifecycleState == lustrefilestorage.LustreFileSystemLifecycleStateDeleting || fs.LifecycleState == lustrefilestorage.LustreFileSystemLifecycleStateDeleted {
		return false
	}
	return true
}

func (f *CloudProviderFramework) WaitForLustreFSDeleted(ctx context.Context, compartmentId, adLocation, pvName string, pollInterval, timeout time.Duration) bool {
	var deleted bool
	err := wait.Poll(pollInterval, timeout, func() (done bool, err error) {
		fsId, err := f.GetLustreFSIdByDisplayName(ctx, compartmentId, adLocation, pvName)
		if err != nil {
			if client.IsNotFound(err) {
				deleted = true
				return true, nil
			}
			return false, err
		}
		exists := f.CheckLustreVolumeExist(ctx, fsId)
		if !exists {
			deleted = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		Logf("Error waiting for Lustre FS deletion: %v", err)
		return false
	}
	return deleted
}
