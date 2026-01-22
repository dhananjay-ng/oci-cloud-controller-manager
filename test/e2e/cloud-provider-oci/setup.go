package e2e

import (
	"strconv"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sharedfw "github.com/oracle/oci-cloud-controller-manager/test/e2e/framework"
	oke "github.com/oracle/oci-go-sdk/v65/containerengine"
	"k8s.io/utils/pointer"
)

var setupF *sharedfw.Framework

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {

	setupF = sharedfw.New()

	sharedfw.Logf("CloudProviderFramework Setup")

	if setupF.EnableCreateCluster {
		createUpgradeTestingNodepool := false
		clusterOCID := ""

		if setupF.IsPostUpgrade {
			clusterOCID = setupF.GetUpgradeTestingCluster(setupF.OkeClusterK8sVersion)
			Expect(clusterOCID).ShouldNot(BeZero())
			sharedfw.Logf("Cluster OCID is %s", clusterOCID)

		} else if setupF.IsPreUpgrade {
			clusterOCID = setupF.GetUpgradeTestingCluster(setupF.OkeClusterK8sVersion)
			if clusterOCID == "" {
				sharedfw.Logf("No cluster with k8s version %s and architecture %s found in Compartment1. Creating new cluster", setupF.OkeClusterK8sVersion, setupF.Architecture)
				createUpgradeTestingNodepool = true
			} else {
				createUpgradeTestingNodepool = !setupF.CheckIfNodepoolExists(clusterOCID)
			}
		}

		if setupF.ExistingClusterOcid != "" {
			clusterOCID = setupF.ExistingClusterOcid
			sharedfw.Logf("Using existing cluster %s ", clusterOCID)
		}

		if clusterOCID == "" {
			sharedfw.Logf("Creating the cluster...")
			clusterOCID = setupF.CreateCluster()
			Expect(clusterOCID).ShouldNot(BeZero())
			sharedfw.Logf("Cluster OCID is %s", clusterOCID)
		}
		setupF.ClusterOcid = clusterOCID
		sharedfw.ClusterID = clusterOCID

		kubeConfig := setupF.CreateClusterKubeconfigContent(clusterOCID)
		Expect(setupF.IsNotJsonFormatStr(kubeConfig)).To(BeTrue())
		Expect(kubeConfig != "").To(BeTrue())

		err := setupF.SaveKubeConfig(kubeConfig)
		Expect(err).NotTo(HaveOccurred())
		sharedfw.Logf("Returned Kubeconfig: \n%s", kubeConfig)

		cloudConfig := setupF.CreateCloudConfig()
		Expect(cloudConfig).ShouldNot(BeNil())

		err = setupF.SaveCloudConfig(cloudConfig)
		Expect(err).Should(BeNil())

		if ((!setupF.IsPreUpgrade && !setupF.IsPostUpgrade) || createUpgradeTestingNodepool) && setupF.ExistingClusterOcid == "" {
			if !setupF.CreateUhpNodepool {
				var ocpus = float32(2.0)
				var memoryInGBs = float32(12.0)
				var NodeShapeConfig = oke.CreateNodeShapeConfigDetails{
					Ocpus:       &ocpus,
					MemoryInGBs: &memoryInGBs,
				}

				size, _ := strconv.Atoi(setupF.NodePoolSize)
				nodepool := setupF.CreateNodePool(clusterOCID, setupF.Compartment1, "Oracle-Linux-7.6",
					setupF.NodeShape, size, setupF.OkeNodePoolK8sVersion,
					[]string{setupF.NodeSubnet, setupF.NodeSubnet, setupF.NodeSubnet},
					NodeShapeConfig, "", nil)
				Expect(nodepool).ShouldNot(BeNil())
				sharedfw.Logf(" Created cluster %s with nodepool %s ", clusterOCID, *nodepool.Id)
			} else {
				var ocpus = float32(16.0)
				var memoryInGBs = float32(16.0)
				var NodeShapeConfig = oke.CreateNodeShapeConfigDetails{
					Ocpus:       &ocpus,
					MemoryInGBs: &memoryInGBs,
				}
				size, _ := strconv.Atoi(setupF.NodePoolSize)
				nodepool := setupF.CreateNodePool(clusterOCID, setupF.Compartment1, "Oracle-Linux-7.6",
					setupF.NodeShape, size, setupF.OkeNodePoolK8sVersion,
					[]string{setupF.NodeSubnet, setupF.NodeSubnet, setupF.NodeSubnet},
					NodeShapeConfig, "", nil)
				Expect(nodepool).ShouldNot(BeNil())
				sharedfw.Logf(" Created cluster %s with nodepool %s ", clusterOCID, *nodepool.Id)
				setupF.EnableBVMPluginOnNodepool(nodepool)
				sharedfw.Logf("Waiting 10 mins for block volume management plugin to be enabled")
				time.Sleep(10 * time.Minute)
			}

			if setupF.EnableLustreTests {
				//Adding new nodepool for lustre tests, this is required because lustre tests require lustre client packages to be installed
				//These packages are not currently available in YUM repos, so new nodepool with custom OL8 image will be created for this.
				var ocpus = float32(2.0)
				var memoryInGBs = float32(16.0)
				var NodeShapeConfig = oke.CreateNodeShapeConfigDetails{
					Ocpus:       &ocpus,
					MemoryInGBs: &memoryInGBs,
				}
				var nodeInitialLabels = []oke.KeyValue{{
					Key:   pointer.String("oci.oraclecloud.com/lustre-client-configured"),
					Value: pointer.String("true"),
				},
				}
				//Adding custom cloud init with taint --kubelet-extra-args "--register-with-taints=dedicated=lustre:NoSchedule, so that other pods are no scheduled on this node"
				setupF.NodeMetadata["user_data"] = "IyEvYmluL2Jhc2gKY3VybCAtLWZhaWwgLUggIkF1dGhvcml6YXRpb246IEJlYXJlciBPcmFjbGUiIC1MMCBodHRwOi8vMTY5LjI1NC4xNjkuMjU0L29wYy92Mi9pbnN0YW5jZS9tZXRhZGF0YS9va2VfaW5pdF9zY3JpcHQgfCBiYXNlNjQgLS1kZWNvZGUgPi92YXIvcnVuL29rZS1pbml0LnNoCmJhc2ggL3Zhci9ydW4vb2tlLWluaXQuc2ggLS1rdWJlbGV0LWV4dHJhLWFyZ3MgIi0tcmVnaXN0ZXItd2l0aC10YWludHM9ZGVkaWNhdGVkPWx1c3RyZTpOb1NjaGVkdWxlIg=="

				size, _ := strconv.Atoi(setupF.NodePoolSize)
				nodepool := setupF.CreateNodePool(clusterOCID, setupF.Compartment1, "",
					setupF.NodeShape, size, setupF.OkeNodePoolK8sVersion,
					[]string{setupF.NodeSubnet, setupF.NodeSubnet, setupF.NodeSubnet},
					NodeShapeConfig, setupF.LustreWorkerNodeImage, nodeInitialLabels)
				Expect(nodepool).ShouldNot(BeNil())
				sharedfw.Logf(" Created cluster %s with Lustre nodepool %s ", clusterOCID, *nodepool.Id)
			}

			setupF.CrossValidateCluster(clusterOCID, setupF.ValidateChildResources)
			if setupF.CustomDriverHandle != "" {
				sharedfw.Logf("Installing custom driver using helm")
				sharedfw.InstallCustomDriver(setupF.ClusterKubeconfigPath, setupF.CustomDriverHandle, setupF.Compartment1, setupF.Vcn)
			}
		}
	} else {
		sharedfw.Logf("Cluster creation skipped. Running tests with existing cluster.")
	}
	if setupF.EnableCertCreation {
		sharedfw.Logf("Cert creation is enabled. Creating a new certificate with auth Id provided.")
		setupF.CertOCID = setupF.GetOrCreateCertificate()
	}
	return nil
}, func(data []byte) {
	setupF = sharedfw.New()
},
)

var _ = ginkgo.SynchronizedAfterSuite(func() {}, func() {
	sharedfw.Logf("Running AfterSuite actions on all node")
	if !setupF.IsPostUpgrade && !setupF.IsPreUpgrade {
		sharedfw.RunCleanupActions()
		if setupF.EnableCreateCluster {
			setupF.CleanAllWithoutWait()
		}
	}
})
