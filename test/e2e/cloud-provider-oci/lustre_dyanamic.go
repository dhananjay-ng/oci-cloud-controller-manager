// Copyright 2021 Oracle and/or its affiliates. All rights reserved.
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

package e2e

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-cloud-controller-manager/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Lustre Dynamic", func() {
	f := framework.NewDefaultFramework("lustre-dynamic-e2e")
	Context("[cloudprovider][storage][csi][lustre][dynamic]", func() {
		It("Should create and delete lustre volumes", func() {
			pvcJig := framework.NewPVCTestJig(f.ClientSet, "csi-lustre-e2e-test")

			parameters := map[string]string{
				"subnetId":           "ocid1.subnet.oc1..example",
				"availabilityDomain": setupF.AdLocation,
				"compartmentId":      setupF.Compartment1,
				"performanceTier":    "MBPS_PER_TB_125",
			}

			scName := f.CreateStorageClassOrFail(framework.ClassOCICSI, setupF.LustreProvisionerName, parameters, pvcJig.Labels, "WaitForFirstConsumer", false, "Delete", nil)

			pvc := pvcJig.CreateAndAwaitPVCOrFailDynamicLustre(f.Namespace.Name, "50Gi", scName, v1.ClaimBound, nil)
			f.VolumeIds = append(f.VolumeIds, pvc.Spec.VolumeName)

			pvcJig.CheckMultiplePodReadWrite(f.Namespace.Name, pvc.Name, false)

			err := pvcJig.DeleteAndAwaitPVC(f.Namespace.Name, pvc.Name)
			Expect(err).NotTo(HaveOccurred())

			deleted := f.WaitForLustreFSDeleted(context.Background(), setupF.Compartment1, setupF.AdLocation, pvc.Spec.VolumeName, framework.Poll, framework.DefaultTimeout)
			Expect(deleted).To(BeTrue(), "Lustre FS was not deleted")
		})
	})
})
