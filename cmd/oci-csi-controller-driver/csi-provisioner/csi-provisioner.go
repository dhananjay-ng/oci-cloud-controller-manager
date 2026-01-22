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

package csiprovisioner

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/component-base/metrics/legacyregistry"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
	"github.com/kubernetes-csi/external-provisioner/v5/pkg/capacity"
	"github.com/kubernetes-csi/external-provisioner/v5/pkg/capacity/topology"
	ctrl "github.com/kubernetes-csi/external-provisioner/v5/pkg/controller"
	"github.com/kubernetes-csi/external-provisioner/v5/pkg/features"
	"github.com/kubernetes-csi/external-provisioner/v5/pkg/owner"
	snapclientset "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csioptions"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	storagelistersv1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	csitranslationlib "k8s.io/csi-translation-lib"

	"github.com/oracle/oci-cloud-controller-manager/pkg/csi/driver"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayInformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
	referenceGrantv1beta1 "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v11/controller"
)

var (
	extraCreateMetadata = false
	defaultFSType       = "ext4"
	version             = "unknown"
	bvCsiDriver         = "BV"
	lustreCsiDriver     = "Lustre"
	capacityThreads     = flag.Uint("capacity-threads", 1, "Number of simultaneously running threads, handling CSIStorageCapacity objects")
	kubeAPIQPS          = flag.Float32("kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	kubeAPIBurst        = flag.Int("kube-api-burst", 10, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")

	kubeAPICapacityQPS   = flag.Float32("kube-api-capacity-qps", 1, "QPS to use for storage capacity updates while communicating with the kubernetes apiserver. Defaults to 1.0.")
	kubeAPICapacityBurst = flag.Int("kube-api-capacity-burst", 5, "Burst to use for storage capacity updates while communicating with the kubernetes apiserver. Defaults to 5.")
	enableNodeDeployment = flag.Bool("node-deployment", false, "Enables deploying the external-provisioner together with a CSI driver on nodes to manage node-local volumes.")
	enableCapacity       = flag.Bool("enable-capacity", false, "This enables producing CSIStorageCapacity objects with capacity information from the driver's GetCapacity call.")
	/*	capacityMode         = func() *capacity.DeploymentMode {
		mode := capacity.DeploymentModeUnset
		flag.Var(&mode, "capacity-controller-deployment-mode", "Setting this enables producing CSIStorageCapacity objects with capacity information from the driver's GetCapacity call. 'central' is currently the only supported mode. Use it when there is just one active provisioner in the cluster.")
		return &mode
	}()*/
	capacityImmediateBinding    = flag.Bool("capacity-for-immediate-binding", false, "Enables producing capacity information for storage classes with immediate binding. Not needed for the Kubernetes scheduler, maybe useful for other consumers or for debugging.")
	capacityPollInterval        = flag.Duration("capacity-poll-interval", time.Minute, "How long the external-provisioner waits before checking for storage capacity changes.")
	capacityOwnerrefLevel       = flag.Int("capacity-ownerref-level", 1, "The level indicates the number of objects that need to be traversed starting from the pod identified by the POD_NAME and POD_NAMESPACE environment variables to reach the owning object for CSIStorageCapacity objects: 0 for the pod itself, 1 for a StatefulSet, 2 for a Deployment, etc.")
	preventVolumeModeConversion = flag.Bool("prevent-volume-mode-conversion", true, "Prevents an unauthorised user from modifying the volume mode when creating a PVC from an existing VolumeSnapshot.")
	nodeDeployment              *ctrl.NodeDeployment
)

// StartCSIProvisioner main function to start CSI Controller Provisioner
func StartCSIProvisioner(csioptions csioptions.CSIOptions, csiDriver driver.CSIDriver) {
	var config *rest.Config
	var err error

	fsType := defaultFSType
	csiAddress := ""
	endpoint := ""
	volumeNamePrefix := csioptions.VolumeNamePrefix
	if string(csiDriver) == bvCsiDriver {
		csiAddress = csioptions.CsiAddress
		endpoint = csioptions.Endpoint
	} else if string(csiDriver) == lustreCsiDriver {
		csiAddress = csioptions.LustreCsiAddress
		volumeNamePrefix = csioptions.LustreVolumeNamePrefix
		fsType = "lustre"
		endpoint = csioptions.LustreEndpoint
		csioptions.Timeout = time.Duration(720) * time.Second // 12 min default timeout, LFS takes ~10 min for smallest filesystem creation
		if v := os.Getenv("LUSTRE_CSI_PROVISIONER_TIMEOUT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				csioptions.Timeout = time.Duration(n) * time.Second
			}
		}
		csioptions.WorkerThreads = uint(10) //default
		if v := os.Getenv("LUSTRE_CSI_PROVISIONER_WORKER_THREADS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				klog.Infof("Using  %v worker threads for CSI provisioner.", uint(n))
				csioptions.WorkerThreads = uint(n)
			}
		}
	} else {
		csiAddress = csioptions.FssCsiAddress
		volumeNamePrefix = csioptions.FssVolumeNamePrefix
		fsType = ""
		endpoint = csioptions.FssEndpoint
	}

	if err := utilfeature.DefaultMutableFeatureGate.SetFromMap(csioptions.FeatureGates); err != nil {
		klog.Fatal(err)
	}

	ctx := context.Background()

	node := os.Getenv("NODE_NAME")
	if *enableNodeDeployment && node == "" {
		klog.Fatal("The NODE_NAME environment variable must be set when using --enable-node-deployment.")
	}

	if csioptions.ShowVersion {
		fmt.Println(os.Args[0], version)
		os.Exit(0)
	}
	klog.Infof("Version: %s", version)

	if csioptions.MetricsAddress != "" && csioptions.HttpEndpoint != "" {
		klog.Error("only one of `--metrics-address` and `--http-endpoint` can be set.")
		os.Exit(1)
	}
	addr := csioptions.MetricsAddress
	if addr == "" {
		addr = csioptions.HttpEndpoint
	}

	// get the KUBECONFIG from env if specified (useful for local/debug cluster)
	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		klog.Infof("Found KUBECONFIG environment variable set, using that..")
		csioptions.Kubeconfig = kubeconfigEnv
	}

	if csioptions.Master != "" || csioptions.Kubeconfig != "" {
		klog.Infof("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(csioptions.Master, csioptions.Kubeconfig)
	} else {
		klog.Infof("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}

	config.QPS = *kubeAPIQPS
	config.Burst = *kubeAPIBurst

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	// snapclientset.NewForConfig creates a new Clientset for VolumesnapshotV1beta1Client
	snapClient, err := snapclientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create snapshot client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	//serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("Error getting server version: %v", err)
	}

	metricsManager := metrics.NewCSIMetricsManager("" /* driverName */)

	klog.V(2).Infof("Creating grpcClient with address %s", csiAddress)
	grpcClient, err := ctrl.Connect(ctx, csiAddress, metricsManager)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	err = ctrl.Probe(ctx, grpcClient, csioptions.OperationTimeout)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// Autodetect provisioner name
	provisionerName, err := ctrl.GetDriverName(grpcClient, csioptions.OperationTimeout)
	if err != nil {
		klog.Fatalf("Error getting CSI driver name: %s", err)
	}
	klog.V(2).Infof("Detected CSI driver %s", provisionerName)
	metricsManager.SetDriverName(provisionerName)

	//metricsManager.StartMetricsEndpoint(csioptions.MetricsAddress, csioptions.MetricsPath)
	// Prepare http endpoint for metrics + leader election healthz
	mux := http.NewServeMux()
	gatherers := prometheus.Gatherers{
		legacyregistry.DefaultGatherer,
		metricsManager.GetRegistry(),
	}

	pluginCapabilities, controllerCapabilities, err := ctrl.GetDriverCapabilities(grpcClient, csioptions.OperationTimeout)
	if err != nil {
		klog.Fatalf("Error getting CSI driver capabilities: %s", err)
	}

	// Generate a unique ID for this provisioner
	timeStamp := time.Now().UnixNano() / int64(time.Millisecond)
	identity := strconv.FormatInt(timeStamp, 10) + "-" + strconv.Itoa(rand.Intn(10000)) + "-" + provisionerName
	if *enableNodeDeployment {
		identity = identity + "-" + node
	}
	factory := informers.NewSharedInformerFactory(clientset, ctrl.ResyncPeriodOfCsiNodeInformer)
	var factoryForNamespace informers.SharedInformerFactory // usually nil, only used for CSIStorageCapacity

	// -------------------------------
	// Listers
	// Create informer to prevent hit the API server for all resource request
	scLister := factory.Storage().V1().StorageClasses().Lister()
	claimLister := factory.Core().V1().PersistentVolumeClaims().Lister()

	var vaLister storagelistersv1.VolumeAttachmentLister
	if controllerCapabilities[csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME] {
		klog.Info("CSI driver supports PUBLISH_UNPUBLISH_VOLUME, watching VolumeAttachments")
		vaLister = factory.Storage().V1().VolumeAttachments().Lister()
	} else {
		klog.Info("CSI driver does not support PUBLISH_UNPUBLISH_VOLUME, not watching VolumeAttachments")
	}

	var gatewayClient gatewayclientset.Interface

	enableCrossNamespaceVolumeDataSource := utilfeature.DefaultFeatureGate.Enabled(features.CrossNamespaceVolumeDataSource)

	if enableCrossNamespaceVolumeDataSource {
		// gatewayclientset.NewForConfig creates a new Clientset for GatewayClient
		gatewayClient, err = gatewayclientset.NewForConfig(config)
		if err != nil {
			klog.Fatalf("Failed to create gateway client: %v", err)
		}
	}
	var refGrantLister referenceGrantv1beta1.ReferenceGrantLister
	var gatewayFactory gatewayInformers.SharedInformerFactory
	if enableCrossNamespaceVolumeDataSource {
		gatewayFactory = gatewayInformers.NewSharedInformerFactory(gatewayClient, ctrl.ResyncPeriodOfReferenceGrantInformer)
		referenceGrants := gatewayFactory.Gateway().V1beta1().ReferenceGrants()
		refGrantLister = referenceGrants.Lister()
	}

	var csiNodeLister storagelistersv1.CSINodeLister
	var nodeLister v1.NodeLister
	if ctrl.SupportsTopology(pluginCapabilities) {
		csiNodeLister = factory.Storage().V1().CSINodes().Lister()
		nodeLister = factory.Core().V1().Nodes().Lister()
	}

	// -------------------------------
	// PersistentVolumeClaims informer
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(csioptions.RetryIntervalStart, csioptions.RetryIntervalMax)
	claimQueue := workqueue.NewNamedRateLimitingQueue(rateLimiter, "claims")
	claimInformer := factory.Core().V1().PersistentVolumeClaims().Informer()

	// Setup options
	provisionerOptions := []func(*controller.ProvisionController) error{
		controller.LeaderElection(false), // Always disable leader election in provisioner lib. Leader election should be done here in the CSI provisioner level instead.
		controller.FailedProvisionThreshold(0),
		controller.FailedDeleteThreshold(0),
		controller.RateLimiter(rateLimiter),
		controller.Threadiness(int(csioptions.WorkerThreads)),
		controller.CreateProvisionedPVLimiter(workqueue.DefaultControllerRateLimiter()),
		controller.ClaimsInformer(claimInformer),
	}

	translator := csitranslationlib.New()

	supportsMigrationFromInTreePluginName := ""
	if translator.IsMigratedCSIDriverByName(provisionerName) {
		supportsMigrationFromInTreePluginName, err = translator.GetInTreeNameFromCSIName(provisionerName)
		if err != nil {
			klog.Fatalf("Failed to get InTree plugin name for migrated CSI plugin %s: %v", provisionerName, err)
		}
		klog.V(2).Infof("Supports migration from in-tree plugin: %s", supportsMigrationFromInTreePluginName)

		// Create a new connection with the metrics manager with migrated label
		metricsManager = metrics.NewCSIMetricsManagerWithOptions(provisionerName,
			// Will be provided via default gatherer.
			metrics.WithProcessStartTime(false),
			metrics.WithMigration())
		migratedGrpcClient, err := ctrl.Connect(ctx, endpoint, metricsManager)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
		grpcClient.Close()
		grpcClient = migratedGrpcClient

		err = ctrl.Probe(ctx, grpcClient, csioptions.OperationTimeout)
		if err != nil {
			klog.Error(err.Error())
			os.Exit(1)
		}
	}
	if supportsMigrationFromInTreePluginName != "" {
		provisionerOptions = append(provisionerOptions, controller.AdditionalProvisionerNames([]string{supportsMigrationFromInTreePluginName}))
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	csiProvisioner := ctrl.NewCSIProvisioner(
		clientset,
		csioptions.OperationTimeout,
		identity,
		volumeNamePrefix,
		csioptions.VolumeNameUUIDLength,
		grpcClient,
		snapClient,
		provisionerName,
		pluginCapabilities,
		controllerCapabilities,
		supportsMigrationFromInTreePluginName,
		csioptions.StrictTopology,
		csioptions.ImmediateTopology,
		translator,
		scLister,
		csiNodeLister,
		nodeLister,
		claimLister,
		vaLister,
		refGrantLister,
		extraCreateMetadata,
		fsType,
		nodeDeployment,
		csioptions.ControllerPublishReadOnly,
		*preventVolumeModeConversion,
	)

	var capacityController *capacity.Controller
	if *enableCapacity {
		// Publishing storage capacity information uses its own client
		// with separate rate limiting.
		config.QPS = *kubeAPICapacityQPS
		config.Burst = *kubeAPICapacityBurst
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			klog.Fatalf("Failed to create client: %v", err)
		}

		namespace := os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			namespace = os.Getenv("NAMESPACE")
		}
		if namespace == "" {
			klog.Fatal("need NAMESPACE env variable for CSIStorageCapacity objects")
		}
		var controller *metav1.OwnerReference
		if *capacityOwnerrefLevel >= 0 {
			podName := os.Getenv("POD_NAME")
			if podName == "" {
				klog.Fatal("need POD_NAME env variable to determine CSIStorageCapacity owner")
			}
			var err error
			controller, err = owner.Lookup(config, namespace, podName,
				schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				}, *capacityOwnerrefLevel)
			if err != nil {
				klog.Fatalf("look up owner(s) of pod: %v", err)
			}
			klog.Infof("using %s/%s %s as owner of CSIStorageCapacity objects", controller.APIVersion, controller.Kind, controller.Name)
		}

		var topologyInformer topology.Informer
		if nodeDeployment == nil {
			topologyInformer = topology.NewNodeTopology(
				provisionerName,
				clientset,
				factory.Core().V1().Nodes(),
				factory.Storage().V1().CSINodes(),
				workqueue.NewNamedRateLimitingQueue(rateLimiter, "csitopology"),
			)
		} else {
			var segment topology.Segment
			if nodeDeployment.NodeInfo.AccessibleTopology != nil {
				for key, value := range nodeDeployment.NodeInfo.AccessibleTopology.Segments {
					segment = append(segment, topology.SegmentEntry{Key: key, Value: value})
				}
			}
			klog.Infof("producing CSIStorageCapacity objects with fixed topology segment %s", segment)
			topologyInformer = topology.NewFixedNodeTopology(&segment)
		}
		go topologyInformer.RunWorker(ctx)

		managedByID := "external-provisioner"
		if *enableNodeDeployment {
			managedByID = getNameWithMaxLength(managedByID, node, validation.DNS1035LabelMaxLength)
		}

		// We only need objects from our own namespace. The normal factory would give
		// us an informer for the entire cluster. We can further restrict the
		// watch to just those objects with the right labels.
		factoryForNamespace = informers.NewSharedInformerFactoryWithOptions(clientset,
			ctrl.ResyncPeriodOfCsiNodeInformer,
			informers.WithNamespace(namespace),
			informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
				lo.LabelSelector = labels.Set{
					capacity.DriverNameLabel: provisionerName,
					capacity.ManagedByLabel:  managedByID,
				}.AsSelector().String()
			}),
		)

		// We use the V1 CSIStorageCapacity API if available.
		clientFactory := capacity.NewV1ClientFactory(clientset)
		cInformer := factoryForNamespace.Storage().V1().CSIStorageCapacities()

		// This invalid object is used in a v1 Create call to determine
		// based on the resulting error whether the v1 API is supported.
		invalidCapacity := &storagev1.CSIStorageCapacity{
			ObjectMeta: metav1.ObjectMeta{
				Name: "#%123-invalid-name",
			},
		}
		createdCapacity, err := clientset.StorageV1().CSIStorageCapacities("default").Create(ctx, invalidCapacity, metav1.CreateOptions{})
		switch {
		case err == nil:
			klog.Fatalf("creating an invalid v1.CSIStorageCapacity didn't fail as expected, got: %s", createdCapacity)
		case apierrors.IsNotFound(err):
			// We need to bridge between the v1beta1 API on the
			// server and the v1 API expected by the capacity code.
			klog.Info("using the CSIStorageCapacity v1beta1 API")
			clientFactory = capacity.NewV1beta1ClientFactory(clientset)
			cInformer = capacity.NewV1beta1InformerBridge(factoryForNamespace.Storage().V1beta1().CSIStorageCapacities())
		case apierrors.IsInvalid(err):
			klog.Info("using the CSIStorageCapacity v1 API")
		default:
			klog.Fatalf("unexpected error when checking for the V1 CSIStorageCapacity API: %v", err)
		}

		capacityController = capacity.NewCentralCapacityController(
			csi.NewControllerClient(grpcClient),
			provisionerName,
			clientFactory,
			// Metrics for the queue is available in the default registry.
			workqueue.NewNamedRateLimitingQueue(rateLimiter, "csistoragecapacity"),
			controller,
			managedByID,
			namespace,
			topologyInformer,
			factory.Storage().V1().StorageClasses(),
			cInformer,
			*capacityPollInterval,
			*capacityImmediateBinding,
			csioptions.OperationTimeout,
		)
		legacyregistry.CustomMustRegister(capacityController)

		// Wrap Provision and Delete to detect when it is time to refresh capacity.
		csiProvisioner = capacity.NewProvisionWrapper(csiProvisioner, capacityController)
	}

	provisionController := newProvisionController(
		ctx,
		csioptions,
		clientset,
		provisionerName,
		csiProvisioner,
		provisionerOptions...,
	)

	csiClaimController := ctrl.NewCloningProtectionController(
		clientset,
		claimLister,
		claimInformer,
		claimQueue,
		controllerCapabilities,
	)
	klog.Infof("HTTP server addr %s", addr)
	// Start HTTP server, regardless whether we are the leader or not.
	if addr != "" {
		// To collect metrics data from the metric handler itself, we
		// let it register itself and then collect from that registry.
		reg := prometheus.NewRegistry()
		gatherers = append(gatherers, reg)

		// This is similar to k8s.io/component-base/metrics HandlerWithReset
		// except that we gather from multiple sources. This is necessary
		// because both CSI metrics manager and component-base manage
		// their own registry. Probably could be avoided by making
		// CSI metrics manager a bit more flexible.
		mux.Handle(csioptions.MetricsPath,
			promhttp.InstrumentMetricHandler(
				reg,
				promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})))
		go func() {
			klog.Infof("ServeMux listening at %q", addr)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				klog.Fatalf("Failed to start HTTP server at specified address (%q) and metrics path (%q): %s", addr, csioptions.MetricsPath, err)
			}
		}()
	}

	run := func(ctx context.Context) {
		factory.Start(ctx.Done())
		if factoryForNamespace != nil {
			// Starting is enough, the capacity controller will
			// wait for sync.
			factoryForNamespace.Start(ctx.Done())
		}
		cacheSyncResult := factory.WaitForCacheSync(ctx.Done())
		for _, v := range cacheSyncResult {
			if !v {
				klog.Fatalf("Failed to sync Informers!")
			}
		}

		if capacityController != nil {
			go capacityController.Run(ctx, int(*capacityThreads))
		}
		if csiClaimController != nil {
			go csiClaimController.Run(ctx, int(csioptions.FinalizerThreads))
		}
		provisionController.Run(ctx)
	}

	if !csioptions.EnableLeaderElection {
		run(ctx)
	} else {
		// this lock name pattern is also copied from sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller
		// to preserve backwards compatibility
		lockName := strings.Replace(provisionerName, "/", "-", -1)

		// create a new clientset for leader election
		leClientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			klog.Fatalf("Failed to create leaderelection client: %v", err)
		}

		le := leaderelection.NewLeaderElection(leClientset, lockName, run)

		if csioptions.HttpEndpoint != "" {
			le.PrepareHealthCheck(mux, leaderelection.DefaultHealthCheckTimeout)
		}

		if csioptions.LeaderElectionNamespace != "" {
			le.WithNamespace(csioptions.LeaderElectionNamespace)
		}

		le.WithIdentity(identity)
		if err := le.Run(); err != nil {
			klog.Fatalf("failed to initialize leader election: %v", err)
		}
	}

}

func newProvisionController(ctx context.Context, csioptions csioptions.CSIOptions, clientset *kubernetes.Clientset, provisionerName string, csiProvisioner controller.Provisioner, provisionerOptions ...func(*controller.ProvisionController) error) *controller.ProvisionController {
	csioptions.RuntimeSchemeMutex.Lock()
	defer csioptions.RuntimeSchemeMutex.Unlock()
	provisionController := controller.NewProvisionController(
		ctx,
		clientset,
		provisionerName,
		csiProvisioner,
		provisionerOptions...,
	)
	return provisionController
}
