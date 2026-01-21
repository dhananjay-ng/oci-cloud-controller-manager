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

var _ = Describe("Lustre E2E Tests", func() {
	f := framework.NewDefaultFramework("lustre-dynamic-e2e")
	Context("[cloudprovider][storage][csi][lustretest]", func() {
		It("All the lustre dynamic provisioning and static provisioning tests", func() {
			if !setupF.EnableLustreTests {
				Skip("Skipping Lustre tests as Lustre tests are not been enabled (Env var: ENABLE_LUSTRE_TESTS)")
			}
			By("Running test: Lustre Dynamic Provisioning - Create lustre file storage dynamically and deploy multiple pods for reading and writing data.")
			pvcJig := framework.NewPVCTestJig(f.ClientSet, "csi-lustre-e2e-test")

			parameters := map[string]string{
				"subnetId":                  setupF.PodSubnet,
				"availabilityDomain":        setupF.AdLocation,
				"performanceTier":           "MBPS_PER_TB_125",
				"kmsKeyId":                  setupF.CMEKKMSKey,
				"setupLnet":                 "true",
				"lustrePostMountParameters": "[{\"*.*.*MDT*.lru_size\" : 11201}]",
			}
			if setupF.LustreSubnetCidr != "" {
				parameters["lustreSubnetCidr"] = setupF.LustreSubnetCidr
			}
			if setupF.CMEKKMSKey != "" {
				parameters["kmsKeyId"] = setupF.CMEKKMSKey
			}

			scName := f.CreateStorageClassOrFail(framework.ClassOCICSI, setupF.LustreProvisionerName, parameters, pvcJig.Labels, "WaitForFirstConsumer", false, "Delete", nil)

			pvc := pvcJig.CreateAndAwaitPVCOrFailDynamicLustre(f.Namespace.Name, "31200Gi", scName, v1.ClaimBound, nil)
			f.VolumeIds = append(f.VolumeIds, pvc.Spec.VolumeName)

			config := &framework.PodConfig{
				NodeSelector: map[string]string{
					"oci.oraclecloud.com/lustre-client-configured": "true",
				},
				Tolerations: []v1.Toleration{
					{
						Key:      "dedicated",
						Operator: v1.TolerationOpEqual,
						Value:    "lustre",
						Effect:   v1.TaintEffectNoSchedule,
					},
				},
			}
			pvcJig.CheckMultiplePodReadWriteGeneric(f.Namespace.Name, pvc.Name, config)
			pv := pvcJig.GetPVByName(pvc.Spec.VolumeName)
			Expect(pv).ToNot(BeNil(), "PV Not present")

			// This dynamically provisioned LFS will be used for additional static provisioning tests due to cost reasons
			setupF.VolumeHandle = pv.Spec.CSI.VolumeHandle
			By("Completed test: Lustre Dynamic Provisioning - Create lustre file storage dynamically and deploy multiple pods for reading and writing data.")

			By("Running test: Lustre Static Provisioning - Multiple pods should be able to read/write to statically provisioned LFS.")
			pvVolumeAttributes := map[string]string{"setupLnet": "true"}
			if setupF.LustreSubnetCidr != "" {
				pvVolumeAttributes["lustreSubnetCidr"] = setupF.LustreSubnetCidr
			}

			pv1 := pvcJig.CreatePVorFailLustre(f.Namespace.Name, setupF.LustreVolumeHandle, []string{}, pvVolumeAttributes)
			pvc1 := pvcJig.CreateAndAwaitPVCOrFailStaticLustre(f.Namespace.Name, pv1.Name, "31200Gi", nil)
			f.VolumeIds = append(f.VolumeIds, pvc1.Spec.VolumeName)
			pvcJig.CheckMultiplePodReadWriteGeneric(f.Namespace.Name, pvc1.Name, config)
			err := pvcJig.DeleteAndAwaitPVC(f.Namespace.Name, pvc1.Name)
			Expect(err).To(BeNil(), "PCV Deletion failed")
			err = pvcJig.DeleteAndAwaitPV(pv1.Name)
			Expect(err).To(BeNil(), "PV Deletion failed")

			By("Complete: Lustre Static Provisioning - Multiple pods should be able to read/write to statically provisioned LFS.")

			By("Running test: Lustre Static Provisioning - Post mount parameters")
			pvVolumeAttributes2 := map[string]string{"setupLnet": "true", "lustrePostMountParameters": "[{\"*.*.*MDT*.lru_size\" : 11201}]"}
			if setupF.LustreSubnetCidr != "" {
				pvVolumeAttributes2["lustreSubnetCidr"] = setupF.LustreSubnetCidr
			}
			pv2 := pvcJig.CreatePVorFailLustre(f.Namespace.Name, setupF.LustreVolumeHandle, []string{}, pvVolumeAttributes2)
			pvc2 := pvcJig.CreateAndAwaitPVCOrFailStaticLustre(f.Namespace.Name, pv2.Name, "31200Gi", nil)
			f.VolumeIds = append(f.VolumeIds, pvc2.Spec.VolumeName)
			pvcJig.CheckSinglePodReadWriteLustre(f.Namespace.Name, pvc.Name, []string{}, true)

			err = pvcJig.DeleteAndAwaitPVC(f.Namespace.Name, pvc2.Name)
			Expect(err).NotTo(HaveOccurred(), "PCV Deletion failed")
			err = pvcJig.DeleteAndAwaitPV(pv2.Name)
			Expect(err).NotTo(HaveOccurred(), "PV Deletion failed")
			By("Complete: Lustre Static Provisioning - Post mount parameters")

			By("Running test: Lustre Static Provisioning - Mount options")
			mountOptions := []string{"flock"}
			pvVolumeAttributes3 := map[string]string{"setupLnet": "true"}
			if setupF.LustreSubnetCidr != "" {
				pvVolumeAttributes3["lustreSubnetCidr"] = setupF.LustreSubnetCidr
			}

			pv3 := pvcJig.CreatePVorFailLustre(f.Namespace.Name, setupF.LustreVolumeHandle, mountOptions, pvVolumeAttributes)
			pvc3 := pvcJig.CreateAndAwaitPVCOrFailStaticLustre(f.Namespace.Name, pv3.Name, "31200Gi", nil)
			f.VolumeIds = append(f.VolumeIds, pvc3.Spec.VolumeName)
			pvcJig.CheckSinglePodReadWriteLustre(f.Namespace.Name, pvc3.Name, mountOptions, false)
			err = pvcJig.DeleteAndAwaitPVC(f.Namespace.Name, pvc3.Name)
			Expect(err).NotTo(HaveOccurred(), "PCV Deletion failed")
			err = pvcJig.DeleteAndAwaitPV(pv3.Name)
			Expect(err).NotTo(HaveOccurred(), "PV Deletion failed")
			By("Complete: Lustre Static Provisioning - Mount Options")

			By("Running test: Lustre Static Provisioning - Applying FS Group ")

			pvVolumeAttributes4 := map[string]string{"setupLnet": "true"}
			if setupF.LustreSubnetCidr != "" {
				pvVolumeAttributes4["lustreSubnetCidr"] = setupF.LustreSubnetCidr
			}

			pv4 := pvcJig.CreatePVorFailLustre(f.Namespace.Name, setupF.LustreVolumeHandle, []string{}, pvVolumeAttributes)
			pvc4 := pvcJig.CreateAndAwaitPVCOrFailStaticLustre(f.Namespace.Name, pv4.Name, "50Gi", nil)

			f.VolumeIds = append(f.VolumeIds, pvc4.Spec.VolumeName)
			pod := pvcJig.CreateAndAwaitNginxPodOrFail(f.Namespace.Name, pvc4, WriteCommand)
			pvcJig.CheckVolumeOwnership(f.Namespace.Name, pod, "/usr/share/nginx/html/", "1000")
			By("Complete: Lustre Static Provisioning - Applying FS Group")

			By("Running test: Delete dynamically created PVC for lustre file storage and make sure its deleted.")
			err = pvcJig.DeleteAndAwaitPVC(f.Namespace.Name, pvc.Name)
			Expect(err).NotTo(HaveOccurred(), "PVC Deletion failed")
			deleted := f.WaitForLustreFSDeleted(context.Background(), setupF.Compartment1, setupF.AdLocation, pvc.Spec.VolumeName, framework.Poll, framework.DefaultTimeout)
			Expect(deleted).To(BeTrue(), "Lustre FS was not deleted")
			By("Completed test: Delete dynamically created PVC for lustre file storage and make sure its deleted.")

		})
	})
})
