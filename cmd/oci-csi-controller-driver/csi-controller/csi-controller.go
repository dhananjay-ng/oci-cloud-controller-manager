// Copyright 2019 Oracle and/or its affiliates. All rights reserved.
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

package csicontroller

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csi-attacher"
	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csi-controller-driver"
	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csi-provisioner"
	csiresizer "github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csi-resizer"
	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csioptions"
	snapshotenabler "github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/snapshot-enabler"
	"github.com/oracle/oci-cloud-controller-manager/pkg/csi/driver"
	"github.com/oracle/oci-cloud-controller-manager/pkg/logging"
)

// Run main function to start CSI Controller
func Run(csioptions csioptions.CSIOptions, stopCh <-chan struct{}) error {
	log := logging.Logger()
	logger := log.Sugar()

	config, err := clientcmd.BuildConfigFromFlags(csioptions.Master, csioptions.Kubeconfig)
	clientset, err := kubernetes.NewForConfig(config)
	err = wait.PollUntil(15*time.Second, func() (done bool, err error) {
		_, err = clientset.Discovery().ServerVersion()
		if err != nil {
			logger.With(zap.Error(err)).Info("failed to get kube-apiserver version, will retry again")
			return false, nil
		}
		return true, nil
	}, stopCh)
	if err != nil {
		return errors.Wrapf(err, "failed to get kube-apiserver version")
	}

	// provisioner for block volume
	logger.Info("starting csi-provisioner go routine for BV")
	go csiprovisioner.StartCSIProvisioner(csioptions, driver.BV)
	//setting operation timeout to 240 seconds for FSS driver (used for CreateVolume/DeleteVolume gRPCs)
	csioptions.OperationTimeout = 240 * time.Second
	// provisioner for fss
	logger.Info("starting csi-provisioner go routine for FSS")
	go csiprovisioner.StartCSIProvisioner(csioptions, driver.FSS)

	// provisioner for lustre (guarded by env flag; default enabled if unset)
	if func() bool { v := os.Getenv("LUSTRE_CSI_CONTROLLER_DRIVER_ENABLED"); return v == "" || strings.EqualFold(v, "true") }() {
		logger.Info("starting csi-provisioner go routine for Lustre")
		go csiprovisioner.StartCSIProvisioner(csioptions, driver.Lustre)
	}

	//setting timeout to 200 seconds for BV driver (used for ControllerPublish/ControllerUnpublish/ControllerExpand gRPCs)
	csioptions.Timeout = 200 * time.Second
	logger.Info("starting csi-attacher go routine for BV")
	go csiattacher.StartCSIAttacher(csioptions)
	if csioptions.EnableResizer {
		logger.Info("starting csi-resizer go routine")
		go csiresizer.StartCSIResizer(csioptions)
	}

	logger = logger.With(zap.String("component", "csi-controller"))

	logger.Info("starting snapshot-enabler go routine for BV")
	go snapshotenabler.Start(csioptions, logger)

	// controller for block volume
	logger.Info("starting csi-controller go routine for BV")
	go csicontrollerdriver.StartControllerDriver(csioptions, driver.BV)

	// controller for fss
	logger.Info("starting csi-controller go routine for FSS")
	go csicontrollerdriver.StartControllerDriver(csioptions, driver.FSS)

	// controller for lustre (guarded by env flag; default enabled if unset)
	if func() bool { v := os.Getenv("LUSTRE_CSI_CONTROLLER_DRIVER_ENABLED"); return v == "" || strings.EqualFold(v, "true") }() {
		logger.Info("starting csi-controller go routine for Lustre")
		go csicontrollerdriver.StartControllerDriver(csioptions, driver.Lustre)
	}

	<-stopCh
	return nil
}
