package framework

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	imageutils "k8s.io/kubernetes/test/utils/image"

	"github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci/config"

	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v65/common"
	oke "github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"gopkg.in/yaml.v2"
)

const (
	// Poll is the default polling period when checking lifecycle status.
	Poll = 15 * time.Second
	// Poll defines how regularly to poll kubernetes resources.
	K8sResourcePoll = 2 * time.Second
	// DefaultTimeout is how long we wait for long-running operations in the
	// test suite before giving up.
	DefaultTimeout = 15 * time.Minute
	// Some pods can take much longer to get ready due to volume attach/detach latency.
	slowPodStartTimeout = 15 * time.Minute

	JobCompletionTimeout       = 5 * time.Minute
	deploymentAvailableTimeout = 5 * time.Minute
	CloneAvailableTimeout      = 10 * time.Minute

	// Expected number of statefulsets to be created during upgrade testing.
	ExpectedStatefulSets = 7

	DefaultClusterKubeconfig = "/tmp/clusterkubeconfig"
	DefaultCloudConfig       = "/tmp/cloudconfig"

	ClassOCI          = "oci"
	ClassOCICSI       = "oci-bv"
	ClassCustom       = "oci-bv-custom"
	ClassOCICSIExpand = "oci-bv-expand"
	ClassOCILowCost   = "oci-bv-low"
	ClassOCIBalanced  = "oci-bal"
	ClassOCIHigh      = "oci-bv-high"
	ClassOCIUHP       = "oci-uhp"
	ClassOCIKMS       = "oci-kms"
	ClassOCIExt3      = "oci-ext3"
	ClassOCIXfs       = "oci-xfs"
	ClassFssDynamic   = "oci-file-storage-test"
	ClassSnapshot     = "oci-snapshot-sc"
	MinVolumeBlock    = "50Gi"
	MaxVolumeBlock    = "100Gi"
	VolumeFss         = "1Gi"

	VSClassDefault    = "oci-snapclass"
	NodeHostnameLabel = "kubernetes.io/hostname"
)

var (
	okeendpoint                   string
	defaultOCIUser                string
	defaultOCIUserFingerprint     string
	defaultOCIUserKeyFile         string
	defaultOCIUserPassphrase      string
	defaultOCITenancy             string
	defaultOCIRegion              string
	compartment1                  string
	vcn                           string
	lbsubnet1                     string
	lbsubnet2                     string
	lbrgnsubnet                   string
	nodeshape                     string
	nodepoolsize                  string
	subnet1                       string
	subnet2                       string
	subnet3                       string
	k8ssubnet                     string
	nodesubnet                    string
	clusterIPFamily               string
	npImageOS                     string
	existingClusterOcid           string
	skipClusterDeletion           string
	okeProvidersK8sVersion        string
	okeClusterK8sVersionIndex     int
	okeNodePoolK8sVersionIndex    int
	pubsshkey                     string
	flexvolumedrivertestversion   string
	volumeprovisionertestversion  string
	secretsDir                    string
	defaultKubeConfig             string
	delegateGroupID               string
	instanceCfg                   *DelegationPrincipalConfig
	enableCreateCluster           bool
	kmsKeyID                      string
	adlocation                    string
	clusterkubeconfig             string // path to kubeconfig file
	deleteNamespace               bool   // whether or not to delete test namespaces
	cloudConfigFile               string // path to cloud provider config file
	nodePortTest                  bool   // whether or not to test the connectivity of node ports.
	ccmSeclistID                  string // The ocid of the loadbalancer subnet seclist. Optional.
	k8sSeclistID                  string // The ocid of the k8s worker subnet seclist. Optional.
	mntTargetOCID                 string // Mount Target ID is specified to identify the mount target to be attached to the volumes. Optional.
	mntTargetSubnetOCID           string // mntTargetSubnetOCID is required for testing MT creation in FSS dynamic
	mntTargetCompartmentOCID      string // mntTargetCompartmentOCID is required for testing MT cross compartment creation in FSS dynamic
	nginx                         string // Image for nginx
	agnhost                       string // Image for agnhost
	busyBoxImage                  string // Image for busyBoxImage
	centos                        string // Image for centos
	imagePullRepo                 string // Repo to pull images from. Will pull public images if not specified.
	cmekKMSKey                    string // KMS key for CMEK testing
	nsgOCIDS                      string // Testing CCM NSG feature
	backendNsgIds                 string // Testing Rule management Backend NSG feature
	reservedIP                    string // Testing public reserved IP feature
	architecture                  string
	volumeHandle                  string // The FSS mount volume handle
	lustreVolumeHandle            string // The Lustre mount volume handle
	lustreSubnetCidr              string // The Lustre Subnet Cidr
	enableLustreTests             bool   // Flag to enable disable lustre tests
	lustreWorkerNodeImage         string // Ocid of worker node image having backed in lustre clients to create separate nodepool
	lustreKMSKey                  string
	staticSnapshotCompartmentOCID string // Compartment ID for cross compartment snapshot test
	customDriverHandle            string // Custom driver handle for custom CSI driver installation
	createUhpNodepool             bool   // Creates UHP nodepool instead of normal nodepool
	namespace                     string // Namespace for pre-upgrade and post-upgrade testing
	isPreUpgradeBool              bool
	isPostUpgradeBool             bool
	isPreUpgradeString            string
	isPostUpgradeString           string
	ClusterID                     string // Ocid of the newly created E2E cluster
	clusterType                   string // Cluster type can be BASIC_CLUSTER or ENHANCED_CLUSTER (Default: BASIC_CLUSTER)
	enableParallelRun             bool
	clusterTypeEnum               oke.ClusterTypeEnum // Enum for OKE Cluster Type
	addOkeSystemTags              bool
	podsubnet                     string
	maxPodsPerNode                int
	cniType                       string
	cniTypeEnum                   oke.ClusterPodNetworkOptionDetailsCniTypeEnum
	nodeMetadata                  map[string]string
	certOcid                      string // OCId of the certificate to be added with LB service
	enableCertificateCreation     bool   // Boolean indicator of the certificate needs to be created. (Currently in same compartment as cluster)
	certAuthorityOcid             string
)

func init() {
	flag.StringVar(&okeendpoint, "okeendpoint", "", "OKE endpoint to test against.")
	flag.StringVar(&defaultOCIUser, "ociuser", "", "OCI user OCID.")
	flag.StringVar(&defaultOCIUserFingerprint, "ocifingerprint", "", "OCI user fingerprint.")
	flag.StringVar(&defaultOCIUserKeyFile, "ocikeyfile", "", "OCI key file.")
	flag.StringVar(&defaultOCIUserPassphrase, "ocikeypassphrase", "", "OCI key passphrase.")
	flag.StringVar(&defaultOCITenancy, "ocitenancy", "", "OCI tenancy.")
	flag.StringVar(&defaultOCIRegion, "ociregion", "", "OCI region.")
	flag.StringVar(&compartment1, "compartment1", "", "OCID of the compartment1 in which to manage clusters.")
	flag.StringVar(&vcn, "vcn", "", "OCID of the VCN in which to create clusters.")
	flag.StringVar(&lbsubnet1, "lbsubnet1", "", "OCID of the 1st subnet in which to create load balancers.")
	flag.StringVar(&lbsubnet2, "lbsubnet2", "", "OCID of the 2nd subnet in which to create load balancers.")
	flag.StringVar(&lbrgnsubnet, "lbrgnsubnet", "", "OCID of the regional subnet in which to create load balancers.")
	flag.StringVar(&nodeshape, "nodeshape", "VM.Standard1.2", "node shape of the nodepool.")
	flag.StringVar(&nodepoolsize, "nodepoolsize", "3", "nodepool size")
	flag.StringVar(&subnet1, "subnet1", "", "OCID of the 1st worker subnet.")
	flag.StringVar(&subnet2, "subnet2", "", "OCID of the 2nd worker subnet.")
	flag.StringVar(&subnet3, "subnet3", "", "OCID of the 3rd worker subnet.")
	flag.StringVar(&k8ssubnet, "k8ssubnet", "", "OCID of the K8s API endpoint subnet.")
	flag.StringVar(&nodesubnet, "nodesubnet", "", "OCID of the nodepool subnet.")
	flag.StringVar(&clusterIPFamily, "clusterIPFamily", "IPv4", "IP Family of cluster to be created.")
	flag.StringVar(&npImageOS, "npImageOS", "", "Node Pool OS Version to be used for testing.")
	flag.StringVar(&existingClusterOcid, "existingClusterOcid", "", "OCID of existing cluster to run e2es on")
	flag.StringVar(&skipClusterDeletion, "skipClusterDeletion", "false", "Flag to control cluster deletion post e2e run, useful to debug cluster in case of issues by skipping deletion.")
	flag.StringVar(&okeProvidersK8sVersion, "okeProvidersK8sVersion", "", "The exact k8s version provided by providers pipeline")
	flag.IntVar(&okeClusterK8sVersionIndex, "okeClusterK8sVersionIndex", -1, "The index of k8s versionList (0 means the 1st version, 1 means the 2nd version. -1 means the latest version. versionList is like ['1.10.11', 1.11.8', '1.12.6']) used when create cluster")
	flag.IntVar(&okeNodePoolK8sVersionIndex, "okeNodePoolK8sVersionIndex", -1, "The index of k8s versionList (0 means the 1st version, 1 means the 2nd version. -1 means the latest version. versionList is like ['1.10.11', 1.11.8', '1.12.6']) used when create nodepool")
	flag.StringVar(&pubsshkey, "pubsshkey", "", "Public SSH Key for node access.")
	flag.StringVar(&flexvolumedrivertestversion, "flexvolumedrivertestversion", "", "FlexVolumeDriver test version.")
	flag.StringVar(&volumeprovisionertestversion, "volumeprovisionertestversion", "", "VolumeProvisioner test version.")
	flag.StringVar(&secretsDir, "secrets-dir", "/secrets", "path to the root secrets directory")
	flag.StringVar(&defaultKubeConfig, "kubeconfig_file", "", "the path to the kubeconfig file for the regional Primordial cluster")
	flag.StringVar(&delegateGroupID, "delegate-group-id", "", "the ocid of the dynamic group to used for fetching delegation tokens")

	flag.BoolVar(&enableCreateCluster, "enable-create-cluster", true, "Whether or not to enable creating a cluster before test execution")

	flag.StringVar(&kmsKeyID, "kms-key-id", "ocid1.key.oc1.iad.annnb3f4aacuu.abuwcljsj6vlwxrtm2bzjmre3ynqfnhkfmcayia3mq47opjp5f2joozrrdaa", "The KMS Key OCID used for testing KMS integration")

	flag.StringVar(&adlocation, "adlocation", "", "Default Ad Location.")

	//Below two flags need to be provided if test cluster already exists.
	flag.StringVar(&clusterkubeconfig, "cluster-kubeconfig", DefaultClusterKubeconfig, "Path to Cluster's Kubeconfig file with authorization and master location information. Only provide if test cluster exists.")
	flag.StringVar(&cloudConfigFile, "cloud-config", DefaultCloudConfig, "The path to the cloud provider configuration file. Empty string for no configuration file. Only provide if test cluster exists.")

	flag.BoolVar(&nodePortTest, "nodeport-test", false, "If true test will include 'nodePort' connectectivity tests.")
	flag.StringVar(&ccmSeclistID, "ccm-seclist-id", "", "The ocid of the loadbalancer subnet seclist. Enables additional seclist rule tests. If specified the 'k8s-seclist-id parameter' is also required.")
	flag.StringVar(&k8sSeclistID, "k8s-seclist-id", "", "The ocid of the k8s worker subnet seclist. Enables additional seclist rule tests. If specified the 'ccm-seclist-id parameter' is also required.")
	flag.BoolVar(&deleteNamespace, "delete-namespace", true, "If true tests will delete namespace after completion. It is only designed to make debugging easier, DO NOT turn it off by default.")

	flag.StringVar(&mntTargetOCID, "mnt-target-id", "", "Mount Target ID is required for creating storage class for FSS dynamic testing")
	flag.StringVar(&mntTargetSubnetOCID, "mnt-target-subnet-id", "", "Mount Target Subnet is required for creating storage class for FSS dynamic testing")
	flag.StringVar(&mntTargetCompartmentOCID, "mnt-target-compartment-id", "", "Mount Target Compartment is required for creating storage class for FSS dynamic testing with cross compartment")
	flag.StringVar(&volumeHandle, "volume-handle", "", "FSS volume handle used to mount the File System")
	flag.StringVar(&lustreVolumeHandle, "lustre-volume-handle", "", "Lustre volume handle used to mount the File System")
	flag.StringVar(&lustreSubnetCidr, "lustre-subnet-cidr", "", "Lustre subnet cidr to identify SVNIC in lustre subnet to configure lnet.")
	flag.BoolVar(&enableLustreTests, "enable-lustre-tests", false, "Flag to control lustre tests.")
	flag.StringVar(&lustreWorkerNodeImage, "lustre-worker-node-image", "", "Worker node image which has lustre clients backed in for creating node pool.")
	flag.StringVar(&lustreKMSKey, "lustre-kms-key", "", "Lustre KMS Key")
	flag.StringVar(&imagePullRepo, "image-pull-repo", "", "Repo to pull images from. Will pull public images if not specified.")
	flag.StringVar(&cmekKMSKey, "cmek-kms-key", "", "KMS key to be used for CMEK testing")
	flag.StringVar(&nsgOCIDS, "nsg-ocids", "", "NSG OCIDs to be used to associate to LB")
	flag.StringVar(&backendNsgIds, "backend-nsg-ocids", "", "backend NSG Ids associated with backends of LB")
	flag.StringVar(&reservedIP, "reserved-ip", "", "Public reservedIP to be used for testing loadbalancer with reservedIP")
	flag.StringVar(&architecture, "architecture", "", "CPU architecture to be used for testing.")

	flag.StringVar(&staticSnapshotCompartmentOCID, "static-snapshot-compartment-id", "", "Compartment ID for cross compartment snapshot test")
	flag.StringVar(&customDriverHandle, "custom-driver-handle", "", "Custom driver handle for custom CSI driver installation")
	flag.BoolVar(&createUhpNodepool, "create-uhp-nodepool", false, "Creates UHP nodepool instead of normal nodepool")

	flag.StringVar(&isPreUpgradeString, "pre-upgrade", "", "If true pre upgrade testing will be done.")
	flag.StringVar(&isPostUpgradeString, "post-upgrade", "", "If true post upgrade testing will be done.")
	flag.StringVar(&namespace, "namespace", "pre-upgrade", "Namespace used for pre-upgrade and post-upgrade testing.")

	flag.StringVar(&clusterType, "cluster-type", "BASIC_CLUSTER", "Cluster type can be BASIC_CLUSTER or ENHANCED_CLUSTER")
	flag.StringVar(&podsubnet, "podsubnet", "", "OCID of the pod subnet in which to create pods for CNI type OCI_VCN_IP_NATIVE")
	flag.IntVar(&maxPodsPerNode, "maxpodspernode", MAX_PODS_PER_NODE, "maxPods per node for OCI_VCN_IP_NATIVE")
	flag.StringVar(&cniType, "cni-type", "FLANNEL_OVERLAY", "CNI type can be FLANNEL_OVERLAY or OCI_VCN_IP_NATIVE")
	flag.BoolVar(&enableParallelRun, "enable-parallel-run", true, "Enables parallel running of test suite")
	flag.BoolVar(&addOkeSystemTags, "add-oke-system-tags", true, "Adds oke system tags to new and existing loadbalancers and storage resources")
	flag.StringVar(&certOcid, "cert-ocid", "", "Certificate OCID to use for Loadbalancer Service listener")
	flag.BoolVar(&enableCertificateCreation, "enable-cert-creation", false, "Whether or Not the test should a new certificate before test run")
	flag.StringVar(&certAuthorityOcid, "cert-authority-ocid", "", "Authority Id to be used to create the Certificate to use for Loadbalancer Service listener")
}

func getDefaultOCIUser() OCIUser {
	// defaultUser details are read from flags and there are cases when this function gets invoked before
	// flag.parse has happened resulting in the variables not being initialized. In such case return an empty user.
	if defaultOCIUser == "" {
		return OCIUser{}
	}
	privateKey, err := ioutil.ReadFile(defaultOCIUserKeyFile)
	if err != nil {
		Failf("Error reading oci private key file '%s' : %s", defaultOCIUserKeyFile, err)
	}
	return OCIUser{
		OCID:                 defaultOCIUser,
		Fingerprint:          defaultOCIUserFingerprint,
		PrivateKey:           privateKey,
		PrivateKeyPassPhrase: defaultOCIUserPassphrase,
	}
}

type AuthType string

const (
	UserAuth    AuthType = "user"
	ServiceAuth AuthType = "service"
)

// MAX_PODS_PER_NODE : If CNI_TYPE is OCI_VCN_NATIVE MAX_PODS_PER_NODE is set to 12
const MAX_PODS_PER_NODE = 12

// Framework is the context of the text execution.
type Framework struct {
	//Note: Reusing framework will not work for OBO requests(check interceptor) with changing users and hence it is recommended to use new framework for each context for obo scenarios.
	computeClient     *core.ComputeClient
	clustersClient    *oke.ContainerEngineClient
	identityClient    *identity.IdentityClient
	certificateClient *certificatesmanagement.CertificatesManagementClient

	context context.Context
	timeout time.Duration

	authType AuthType

	RegionalKubeConfig string

	// Set of headers those would be injected in to an outgoing HTTP request through an interceptor.
	requestHeaders map[string]string

	// The OCI tenancy.
	Tenancy string
	// The OCI region.
	Region string

	// The OCI test user.
	User OCIUser

	// The compartment1 the cluster is running in.
	Compartment1 string
	// The VCN the cluster is running in.
	Vcn string
	// Loadbalancer subnet 1.
	LbSubnet1 string
	// Loadbalancer subnet 2.
	LbSubnet2 string
	// Loadbalancer regional subnet
	Lbrgnsubnet string
	// NodePool node shape.
	NodeShape string
	// NodePool size
	NodePoolSize string
	// NodePool subnet 1.
	Subnet1 string
	// NodePool subnet 2.
	Subnet2 string
	// NodePool subnet 3.
	Subnet3 string
	// K8s API endpoint regional subnet
	K8sSubnet string
	// Nodepool subnet
	NodeSubnet string

	ClusterIPFamily string
	//Node Pool Image to be used
	NpImageOS string

	ExistingClusterOcid string

	SkipClusterDeletion string
	// Cluster Type
	ClusterType oke.ClusterTypeEnum

	//k8s version value (eg. v1.10.11, v1.11.8) used when create cluster
	OkeClusterK8sVersion string
	//k8s version value (eg. v1.10.11, v1.11.8) used when create nodepool
	OkeNodePoolK8sVersion string
	// NodePool metadata
	NodeMetadata map[string]string

	// Pod subnet
	PodSubnet string
	// Max pods per node
	MaxPodsPerNode int
	// OCI_VCN_IP_NATIVE or FLANNEL_OVERLAY
	CniType oke.ClusterPodNetworkOptionDetailsCniTypeEnum

	// The Public SSH key to use when accessing nodes.
	PubSSHKey string

	// The lower of the Kubernetes versions supported.
	K8sVersion1 string
	// The middle Kubernetes versions supported.
	K8sVersion2 string
	// The higher of the Kubernetes versions supported.
	K8sVersion3 string
	// The lower of the nodePool Kubernetes versions supported.
	NodePoolK8sVersion1 string
	// The middle nodePool Kubernetes versions supported.
	NodePoolK8sVersion2 string
	// The higher of the nodePool Kubernetes versions supported.
	NodePoolK8sVersion3 string
	// If true, then delete operations will block until the target resource has been deleted.
	WaitForDeleted bool
	// If true, then validation operations should cascade and deleted any dependent child resources.
	ValidateChildResources bool

	// OKEEndpoint is the endpoint url for the TM API we'll be performing tests against.
	OKEEndpoint string

	//DynamicGroup for auth delegation testing
	DynamicGroup       string
	instanceConfigFile string
	// Target services for obtaining delegation token.
	DelegationTargetServices string

	// Default adLocation
	AdLocation string

	// Default adLocation
	AdLabel string

	//is cluster creation required
	EnableCreateCluster bool

	ClusterKubeconfigPath string

	CloudConfigPath string

	MntTargetOcid            string
	MntTargetSubnetOcid      string
	MntTargetCompartmentOcid string
	CMEKKMSKey               string
	NsgOCIDS                 string
	BackendNsgOcid           string
	ReservedIP               string
	Architecture             string

	VolumeHandle       string
	LustreVolumeHandle string

	LustreSubnetCidr      string
	EnableLustreTests     bool
	LustreWorkerNodeImage string
	LustreKMSKey          string

	// Compartment ID for cross compartment snapshot test
	StaticSnapshotCompartmentOcid string
	CustomDriverHandle            string
	BlockProvisionerName          string
	FSSProvisionerName            string
	LustreProvisionerName         string
	CreateUhpNodepool             bool

	UpgradeTestingNamespace string
	IsPreUpgrade            bool
	IsPostUpgrade           bool
	ClusterOcid             string
	AddOkeSystemTags        bool
	CertOCID                string
	EnableCertCreation      bool
	CertAuthorityOCID       string
	KMSKeyOCIDForCA         string
}

// New creates a new a framework that holds the context of the test
// execution.
func New() *Framework {
	flag.Parse()
	return NewWithConfig(&FrameworkConfig{})
}

// FrameworkConfig helps in passing the configuration options while creating a Framework instance.
type FrameworkConfig struct {
	// authType specifies the framework's authType to be used before initialization.
	AuthType AuthType
	// User specifies the framework's OCIUser to be used before initialization.
	User *OCIUser

	// InstanceConfigFile specifies the framework's  InstanceConfigFile to be used before initialization.
	InstanceConfigFile string
}

// OCIUser contains details about an OCI user along with their API key details.
type OCIUser struct {
	// OCID of the User
	OCID string
	// Fingerprint of the public key attached to the User
	Fingerprint string
	// User's API Private key
	PrivateKey []byte
	// PassPhrase of the API Private key. If API Private key has no passphrase then set it to ""
	PrivateKeyPassPhrase string
}

// NewWithConfig creates a new Framework instance and configures the instance as per the configuration options in the given config.
func NewWithConfig(config *FrameworkConfig) *Framework {
	rand.Seed(time.Now().UTC().UnixNano())

	f := &Framework{
		// Default to user auth for new frameworks.
		authType: UserAuth,

		computeClient:  nil,
		clustersClient: nil,

		Tenancy:      defaultOCITenancy,
		Region:       defaultOCIRegion,
		User:         getDefaultOCIUser(),
		Compartment1: compartment1,

		RegionalKubeConfig: defaultKubeConfig,
		requestHeaders:     map[string]string{},

		timeout:                       3 * time.Minute,
		context:                       context.Background(),
		WaitForDeleted:                false,
		ValidateChildResources:        true,
		PubSSHKey:                     pubsshkey,
		Vcn:                           vcn,
		LbSubnet1:                     lbsubnet1,
		LbSubnet2:                     lbsubnet2,
		Lbrgnsubnet:                   lbrgnsubnet,
		Subnet1:                       subnet1,
		Subnet2:                       subnet2,
		Subnet3:                       subnet3,
		K8sSubnet:                     k8ssubnet,
		NodeSubnet:                    nodesubnet,
		ClusterIPFamily:               clusterIPFamily,
		NpImageOS:                     npImageOS,
		ExistingClusterOcid:           existingClusterOcid,
		SkipClusterDeletion:           skipClusterDeletion,
		NodeShape:                     nodeshape,
		NodePoolSize:                  nodepoolsize,
		DelegationTargetServices:      "oke",
		AdLocation:                    adlocation,
		MntTargetOcid:                 mntTargetOCID,
		MntTargetSubnetOcid:           mntTargetSubnetOCID,
		MntTargetCompartmentOcid:      mntTargetCompartmentOCID,
		CMEKKMSKey:                    cmekKMSKey,
		NsgOCIDS:                      nsgOCIDS,
		ReservedIP:                    reservedIP,
		Architecture:                  architecture,
		VolumeHandle:                  volumeHandle,
		LustreVolumeHandle:            lustreVolumeHandle,
		LustreSubnetCidr:              lustreSubnetCidr,
		LustreWorkerNodeImage:         lustreWorkerNodeImage,
		LustreKMSKey:                  lustreKMSKey,
		EnableLustreTests:             enableLustreTests,
		StaticSnapshotCompartmentOcid: staticSnapshotCompartmentOCID,
		CustomDriverHandle:            customDriverHandle,
		CreateUhpNodepool:             createUhpNodepool,
		UpgradeTestingNamespace:       namespace,
		ClusterType:                   clusterTypeEnum,
		AddOkeSystemTags:              addOkeSystemTags,
		ClusterOcid:                   ClusterID,
		CniType:                       cniTypeEnum,
		NodeMetadata:                  nodeMetadata,
		CertOCID:                      certOcid,
		EnableCertCreation:            enableCertificateCreation,
		CertAuthorityOCID:             certAuthorityOcid,
		KMSKeyOCIDForCA:               kmsKeyID,
	}

	f.EnableCreateCluster = enableCreateCluster

	if enableCreateCluster {
		if config.AuthType != "" {
			f.authType = config.AuthType
		}

		if config.InstanceConfigFile != "" {
			f.instanceConfigFile = config.InstanceConfigFile
		}

		if config.User != nil {
			f.User = *config.User
		}

		f.OKEEndpoint = okeendpoint
		if !strings.HasPrefix(okeendpoint, "https://") {
			f.OKEEndpoint = "https://" + okeendpoint
		}

	}

	f.CloudConfigPath = cloudConfigFile
	f.ClusterKubeconfigPath = clusterkubeconfig

	f.Initialize()

	return f
}

func (f *Framework) AddRequestHeader(name string, value string) *Framework {
	f.requestHeaders[name] = value
	return f
}

func (f *Framework) newOCIRequestSigner(authType AuthType) (common.HTTPRequestSigner, error) {

	switch authType {
	case UserAuth:
		cfgProvider := common.NewRawConfigurationProvider(
			f.Tenancy,
			f.User.OCID,
			f.Region,
			f.User.Fingerprint,
			string(f.User.PrivateKey),
			&f.User.PrivateKeyPassPhrase,
		)
		return common.DefaultRequestSigner(cfgProvider), nil
	default:
		return nil, fmt.Errorf("unknown auth type %q: cannot construct signing client", f.authType)
	}
}

// BeforeEach will be executed before each Ginkgo test is executed.
func (f *Framework) Initialize() {
	Logf("initializing framework")
	f.EnableCreateCluster = enableCreateCluster
	f.AdLocation = adlocation
	Logf("OCI AdLocation: %s", f.AdLocation)
	if adlocation != "" {
		splitString := strings.Split(adlocation, ":")
		if len(splitString) == 2 {
			f.AdLabel = splitString[1]
		} else {
			Failf("Invalid Availability Domain %s. Expecting format: `Uocm:PHX-AD-1`", adlocation)
		}
	}
	Logf("OCI AdLabel: %s", f.AdLabel)
	f.MntTargetOcid = mntTargetOCID
	Logf("OCI Mount Target OCID: %s", f.MntTargetOcid)
	f.MntTargetSubnetOcid = mntTargetSubnetOCID
	Logf("OCI Mount Target Subnet OCID: %s", f.MntTargetSubnetOcid)
	f.MntTargetCompartmentOcid = mntTargetCompartmentOCID
	Logf("OCI Mount Target Compartment OCID: %s", f.MntTargetCompartmentOcid)
	f.VolumeHandle = volumeHandle
	Logf("FSS Volume Handle is : %s", f.VolumeHandle)
	f.LustreVolumeHandle = lustreVolumeHandle
	Logf("Lustre Volume Handle is : %s", f.LustreVolumeHandle)
	f.LustreSubnetCidr = lustreSubnetCidr
	Logf("Lustre Subnet CIDR is : %s", f.LustreSubnetCidr)
	f.EnableLustreTests = enableLustreTests
	Logf("EnableLustreTests is : %s", f.EnableLustreTests)
	f.LustreWorkerNodeImage = lustreWorkerNodeImage
	Logf("LustreWorkerNodeImage : %s", f.LustreWorkerNodeImage)
	f.LustreKMSKey = lustreKMSKey
	Logf("LustreKMSKey : %s", f.LustreKMSKey)

	f.StaticSnapshotCompartmentOcid = staticSnapshotCompartmentOCID
	Logf("Static Snapshot Compartment OCID: %s", f.StaticSnapshotCompartmentOcid)
	f.CustomDriverHandle = customDriverHandle
	Logf("Custom CSI Driver Handle: %s", f.CustomDriverHandle)
	f.BlockProvisionerName = getBlockProvisionerName(customDriverHandle)
	Logf("Block Provisioner name: %s", f.BlockProvisionerName)
	f.FSSProvisionerName = getFSSProvisionerName(customDriverHandle)
	Logf("FSS Provisioner name: %s", f.FSSProvisionerName)
	f.LustreProvisionerName = getLustreProvisionerName(customDriverHandle)
	Logf("Lustre Provisioner name: %s", f.LustreProvisionerName)
	f.CreateUhpNodepool = createUhpNodepool
	Logf("Create Uhp Nodepool: %v", f.CreateUhpNodepool)
	f.CMEKKMSKey = cmekKMSKey
	Logf("CMEK KMS Key: %s", f.CMEKKMSKey)
	f.NsgOCIDS = nsgOCIDS
	Logf("NSG OCIDS: %s", f.NsgOCIDS)
	f.BackendNsgOcid = backendNsgIds
	Logf("Backend NSG OCIDS: %s", f.BackendNsgOcid)
	f.ReservedIP = reservedIP
	Logf("Reserved IP: %s", f.ReservedIP)
	f.Architecture = architecture
	Logf("Architecture: %s", f.Architecture)
	f.Compartment1 = compartment1
	Logf("OCI compartment1 OCID: %s", f.Compartment1)
	f.setImages()

	f.PodSubnet = podsubnet
	Logf("OCI pod subnet OCID: %s", f.PodSubnet)

	if !enableCreateCluster {
		Logf("Cluster Creation Disabled")
		f.ClusterKubeconfigPath = clusterkubeconfig
		f.CloudConfigPath = cloudConfigFile
		return
	}
	Logf("Cluster Creation Enabled")
	Logf("Auth Type: %v", f.authType)
	f.PubSSHKey = pubsshkey
	Logf("Public SSHKey : %s", f.PubSSHKey)
	f.Vcn = vcn
	Logf("OCI VCN OCID: %s", f.Vcn)
	f.LbSubnet1 = lbsubnet1
	Logf("OCI lbSubnet1 OCID: %s", f.LbSubnet1)
	f.LbSubnet2 = lbsubnet2
	Logf("OCI lbSubnet2 OCID: %s", f.LbSubnet2)
	f.Lbrgnsubnet = lbrgnsubnet
	Logf("OCI lbrgnsubnet OCID: %s", f.Lbrgnsubnet)
	f.Subnet1 = subnet1
	Logf("OCI Subnet1 OCID: %s", f.Subnet1)
	f.Subnet2 = subnet2
	Logf("OCI Subnet2 OCID: %s", f.Subnet2)
	f.Subnet3 = subnet3
	Logf("OCI Subnet3 OCID: %s", f.Subnet3)
	f.K8sSubnet = k8ssubnet
	Logf("OCI K8sSubnet OCID: %s", f.K8sSubnet)
	f.NodeSubnet = nodesubnet
	Logf("OCI NodeSubnet OCID: %s", f.NodeSubnet)
	f.ClusterIPFamily = clusterIPFamily
	Logf("ClusterIPFamily : %s", f.ClusterIPFamily)
	f.NpImageOS = npImageOS
	Logf("NpImageOS: %s", f.NpImageOS)
	f.ExistingClusterOcid = existingClusterOcid
	Logf("ExistingClusterOcid: %s", f.ExistingClusterOcid)
	f.SkipClusterDeletion = skipClusterDeletion
	Logf("SkipClusterDeletion: %s", f.SkipClusterDeletion)
	f.NodeShape = nodeshape
	Logf("Nodepool NodeShape: %s", f.NodeShape)
	f.NodePoolSize = nodepoolsize
	Logf("Nodepool size: %s", f.NodePoolSize)
	f.AddOkeSystemTags = addOkeSystemTags

	f.NodeMetadata = map[string]string{
		"areLegacyImdsEndpointsDisabled": "true",
	}
	Logf("Using legacyImdsEndpointDisabled %s", f.NodeMetadata["areLegacyImdsEndpointsDisabled"])

	Logf("AddOkeSystemTags : %v", f.AddOkeSystemTags)
	if strings.ToUpper(clusterType) == "ENHANCED_CLUSTER" {
		clusterTypeEnum = oke.ClusterTypeEnhancedCluster
	} else {
		clusterTypeEnum = oke.ClusterTypeBasicCluster
	}
	f.ClusterType = clusterTypeEnum
	Logf("Cluster Type: %s", f.ClusterType)

	f.MaxPodsPerNode = maxPodsPerNode
	Logf("Max pods per node: %s", f.MaxPodsPerNode)

	var err error
	if isPreUpgradeString != "" {
		isPreUpgradeBool, err = strconv.ParseBool(isPreUpgradeString)
		Expect(err).NotTo(HaveOccurred())
	} else {
		isPreUpgradeBool = false
	}
	f.IsPreUpgrade = isPreUpgradeBool
	Logf("Pre-upgrade testing flag: %v, %v", f.IsPreUpgrade, isPreUpgradeBool)

	if isPostUpgradeString != "" {
		isPostUpgradeBool, err = strconv.ParseBool(isPostUpgradeString)
	} else {
		isPostUpgradeBool = false
	}
	f.IsPostUpgrade = isPostUpgradeBool
	Logf("Post-upgrade testing flag: %v, %v", f.IsPostUpgrade, isPostUpgradeBool)

	f.UpgradeTestingNamespace = namespace
	Logf("Upgrade testing namespace: %s", namespace)

	f.OKEEndpoint = okeendpoint
	if !strings.HasPrefix(okeendpoint, "https://") {
		f.OKEEndpoint = "https://" + okeendpoint
	}

	Logf("OCI User: %v", f.User.OCID)
	Logf("OCI finger print: %v", f.User.Fingerprint)

	if f.clustersClient == nil {
		Logf("Creating OCI client")
		//if it's a delegation obo scenario
		if f.instanceConfigFile != "" {
			Logf("instanceConfigFile is %s", f.instanceConfigFile)
			cfgBytes, err := ioutil.ReadFile(f.instanceConfigFile)
			if err != nil {
				Failf("Error reading '%s' : %s", f.instanceConfigFile, err)
			}

			var cfg SecretValuesConfig

			if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
				Failf("Error unmarshaling '%s' : %s", f.instanceConfigFile, err)
			}

			instanceCfg = &cfg.BMCSCredentials.DelegationPrincipalConfig
			f.DynamicGroup = instanceCfg.DynamicGroup
		} else {
			f.DynamicGroup = delegateGroupID
		}

		userConfigProvider := common.NewRawConfigurationProvider(
			f.Tenancy, f.User.OCID, f.Region, f.User.Fingerprint,
			string(f.User.PrivateKey), &f.User.PrivateKeyPassPhrase)

		clustersClient, err := oke.NewContainerEngineClientWithConfigurationProvider(userConfigProvider)

		Expect(err).NotTo(HaveOccurred())
		f.clustersClient = &clustersClient
		f.clustersClient.Host = okeendpoint
		f.clustersClient.BasePath = "20180222"
		f.clustersClient.Interceptor = nil

		requestSigner, err := f.newOCIRequestSigner(f.authType)
		if err != nil {
			Failf("f.authType: %q., unable to create request signer: %v", f.authType, err)
		}
		f.clustersClient.BaseClient.Signer = requestSigner

		Logf("oke endpoint: '%s'", okeendpoint)
		computeClient, err := core.NewComputeClientWithConfigurationProvider(userConfigProvider)
		Expect(err).NotTo(HaveOccurred())
		f.computeClient = &computeClient

		identityClient, err := identity.NewIdentityClientWithConfigurationProvider(userConfigProvider)
		Expect(err).NotTo(HaveOccurred())
		f.identityClient = &identityClient

		// Register a certificate management client
		certificatesClient, err := certificatesmanagement.NewCertificatesManagementClientWithConfigurationProvider(userConfigProvider)
		Expect(err).NotTo(HaveOccurred())
		f.certificateClient = &certificatesClient

		// Determine which cluster k8s versions are supported for this OKE Release
		clusterOptions := f.GetClusterOptions("all")
		versions := filterVersionsToLatestMinor(clusterOptions.KubernetesVersions)
		if okeProvidersK8sVersion != "" {
			Logf("okeProvidersK8sVersion is provided and not empty=%v", okeProvidersK8sVersion)
			f.OkeClusterK8sVersion = okeProvidersK8sVersion
		} else {
			f.OkeClusterK8sVersion = getK8sVersionValue(versions, okeClusterK8sVersionIndex)
		}
		// f.OkeClusterK8sVersion = getK8sVersionValue(versions, okeClusterK8sVersionIndex)
		//below K8sVersion* would be deprecated if OkeClusterK8sVersion is used in test code
		numVersions := len(versions)
		if numVersions < 2 {
			Failf("less than two supported cluster k8sVersions!!!  '%#v'", versions)
		}
		f.K8sVersion1 = versions[0]
		f.K8sVersion2 = versions[1]
		if numVersions >= 3 {
			f.K8sVersion3 = versions[2]
		} else {
			f.K8sVersion3 = ""
		}

		// Determine which cluster k8s versions are supported for this OKE Release
		nodePoolOptions := f.GetNodePoolOptions("all")
		nodePoolVersions := filterVersionsToLatestMinor(nodePoolOptions.KubernetesVersions)
		if okeProvidersK8sVersion != "" {
			Logf("okeProvidersK8sVersion is provided and not empty, hence setting nodepool version as =%v", okeProvidersK8sVersion)
			f.OkeNodePoolK8sVersion = okeProvidersK8sVersion
		} else {
			f.OkeNodePoolK8sVersion = getK8sVersionValue(nodePoolVersions, okeNodePoolK8sVersionIndex)
		}
		//f.OkeNodePoolK8sVersion = getK8sVersionValue(nodePoolVersions, okeNodePoolK8sVersionIndex)

		Logf("OkeClusterK8sVersion=%v", f.OkeClusterK8sVersion)
		Logf("OkeNodePoolK8sVersion=%v", f.OkeNodePoolK8sVersion)
		if CompareVersions(f.OkeClusterK8sVersion, f.OkeNodePoolK8sVersion) < 0 {
			Failf("Cluster K8s Version is less than Nodepool K8s version")
		}

		//below NodePoolK8sVersion* would be deprecated if OkeNodePoolK8sVersion is used in test code
		numNodePoolVersions := len(nodePoolVersions)
		if numNodePoolVersions < 2 {
			Failf("less than two supported node k8sVersions!!!  '%#v'", nodePoolVersions)
		}
		f.NodePoolK8sVersion1 = nodePoolVersions[0]
		f.NodePoolK8sVersion2 = nodePoolVersions[1]
		if numVersions >= 3 {
			f.NodePoolK8sVersion3 = nodePoolVersions[2]
		} else {
			f.NodePoolK8sVersion3 = ""
		}
	}

	if existingClusterOcid != "" {
		type clusterpodnetworkoptiondetails struct {
			JsonData []byte
			CniType  string `json:"cniType"`
		}
		cluster := f.GetCluster(existingClusterOcid)

		switch cluster.ClusterPodNetworkOptions[0].(type) {
		case oke.FlannelOverlayClusterPodNetworkOptionDetails:
			cniType = "FLANNEL_OVERLAY"
		case oke.OciVcnIpNativeClusterPodNetworkOptionDetails:
			cniType = "OCI_VCN_IP_NATIVE"
		}
	}

	if strings.ToUpper(cniType) == "OCI_VCN_IP_NATIVE" && podsubnet != "" {
		cniTypeEnum = oke.ClusterPodNetworkOptionDetailsCniTypeOciVcnIpNative
	} else {
		cniTypeEnum = oke.ClusterPodNetworkOptionDetailsCniTypeFlannelOverlay
	}
	f.CniType = cniTypeEnum
	Logf("CNI Type: %s", f.CniType)

	if maxPodsPerNode == 0 {
		maxPodsPerNode = MAX_PODS_PER_NODE
	}
	Logf("maxPodsPerNode is: %s", f.MaxPodsPerNode)
	f.CertOCID = certOcid
	f.EnableCertCreation = enableCertificateCreation
	if f.EnableCertCreation {
		f.CertAuthorityOCID = certAuthorityOcid
		f.KMSKeyOCIDForCA = kmsKeyID
		Logf("CertAuthorityOCID is: %s", f.CertAuthorityOCID)
	}
	Logf("CertOCID: %s, EnableCertificateCreation : %s", f.CertOCID, f.EnableCertCreation)
}

// getK8sVersionValue returns the version value according to version index
// eg. versions=["1.10.11", "1.11.5"]
// return 1.10.11 if index=0,
// return 1.11.5 if index=1, or -1
func getK8sVersionValue(versions []string, index int) string {
	numVersions := len(versions)
	if numVersions == 0 {
		Failf("No supported k8sVersions for cluster or nodepool!!! ")
	}

	if index < 0 && numVersions+index >= 0 {
		return versions[numVersions+index]
	}

	if index >= 0 && index < numVersions {
		return versions[index]
	}

	Failf("Index=%d is not valid for current version List=%v", index, versions)
	return ""
}

func filterVersionsToLatestMinor(versions []string) []string {
	var filtered []string
	for _, v := range versions {
		if len(filtered) == 0 {
			filtered = append(filtered, v)
			continue
		}

		idx := -1
		for i, existing := range filtered {
			if stripPatchVersion(existing) == stripPatchVersion(v) {
				idx = i
				break
			}
		}

		if idx != -1 {
			filtered[idx] = v
		} else {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

func stripPatchVersion(version string) string {
	parts := strings.Split(version, ".")
	return parts[0] + "." + parts[1]
}

func (f *Framework) CleanAllWithoutWait() []byte {
	f.CleanAll(false)
	return []byte{}
}

// CleanAll will attempt to clean the tenancy and compartment1 of all
// clusters and nodepools. Use with caution.
func (f *Framework) CleanAll(waitForDeleted bool) {
	Logf("Cleaning all OKE clusters in compartment1 '%s' ", f.Compartment1)
	for _, cluster := range f.ListClusters() {
		if cluster.LifecycleState == oke.ClusterSummaryLifecycleStateActive || cluster.LifecycleState == oke.ClusterSummaryLifecycleStateFailed {
			f.DeleteCluster(*cluster.Id, waitForDeleted)
		}
	}
	Logf("Cleaning invalid cert authority in compartment1 '%s' ", f.Compartment1)
	if f.CertAuthorityOCID != "" {
		_, state := f.GetCA()
		if state != certificatesmanagement.CertificateAuthorityLifecycleStateActive {
			f.DeleteCertificateAuthority()
		}
	}

}

// CreateClusterKubeconfigContent gets a valid 'kubeconfig' file for the target
// cluster.
func (f *Framework) CreateClusterKubeconfigContent(clusterID string) string {
	Logf("Getting cluster kubeconfig, id: %s", clusterID)
	ctx, cancel := context.WithTimeout(f.context, f.timeout)

	// pass in the token version to request a v2 kubeconfig
	tokenVersion := "2.0.0"

	defer cancel()
	response, err := f.clustersClient.CreateKubeconfig(ctx, oke.CreateKubeconfigRequest{
		ClusterId:                             &clusterID,
		CreateClusterKubeconfigContentDetails: oke.CreateClusterKubeconfigContentDetails{TokenVersion: &tokenVersion},
	})

	if err != nil {
		Logf("CreateKubeconfig error:  %s; retrying...", err)
		response, err = f.clustersClient.CreateKubeconfig(ctx, oke.CreateKubeconfigRequest{
			ClusterId: &clusterID,
		})
		if err != nil {
			Logf("CreateKubeconfig error:  %s; returning empty content", err)
			return ""
		}
	}

	content, err := ioutil.ReadAll(response.Content)
	if err != nil {
		Logf("ReadContent error:  %s; continuing", err)
	}
	//	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

// GetWorkRequest gets the specified WorkRequest for an initialising cluster.
// Retries added to catch rare connectivity errors
func (f *Framework) GetWorkRequest(id string) oke.GetWorkRequestResponse {
	// retries added because of the occasional connection errors
	timeout := 10 * time.Minute
	ctx, cancel := context.WithTimeout(f.context, f.timeout)
	defer cancel()
	response, err := f.clustersClient.GetWorkRequest(ctx, oke.GetWorkRequestRequest{
		WorkRequestId: &id,
	})
	if err == nil {
		Expect(err).NotTo(HaveOccurred())
		return response
	}

	Logf("Error received on Get workRequest : '%s', Retrying", err)

	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		response, err = f.clustersClient.GetWorkRequest(ctx, oke.GetWorkRequestRequest{
			WorkRequestId: &id,
		})
		if err == nil {
			Expect(err).NotTo(HaveOccurred())
			return response
		}

		Logf("Error received on Get workRequest : '%s', Retrying", err)
	}
	Failf("Timeout waiting for get workRequest '%s'", id)
	return response
}

func (f *Framework) CreateCloudConfig() config.Config {
	cloudConfig := config.Config{
		Auth: config.AuthConfig{
			Region:                f.Region,
			TenancyID:             f.Tenancy,
			UserID:                f.User.OCID,
			PrivateKey:            string(f.User.PrivateKey),
			Fingerprint:           f.User.Fingerprint,
			Passphrase:            f.User.PrivateKeyPassPhrase,
			RegionKey:             "",
			UseInstancePrincipals: false,
			CompartmentID:         "",
			PrivateKeyPassphrase:  "",
		},
		LoadBalancer: &config.LoadBalancerConfig{
			DisableSecurityListManagement: true,
			Subnet1:                       f.LbSubnet1,
			Subnet2:                       f.LbSubnet2,
		},
		RateLimiter:           nil,
		RegionKey:             "",
		UseInstancePrincipals: false,
		CompartmentID:         f.Compartment1,
		VCNID:                 f.Vcn,
		UseServicePrincipals:  false,
	}
	return cloudConfig
}

func (f *Framework) SaveCloudConfig(cloudConfig config.Config) error {
	cloudConfigBytes, err := yaml.Marshal(&cloudConfig)
	if err != nil {
		return err
	}
	cloudConfigFile, err := os.Create(f.CloudConfigPath)
	if err != nil {
		return err
	}
	defer cloudConfigFile.Close()
	bytesWritten, err := cloudConfigFile.Write(cloudConfigBytes)
	if err != nil {
		return err
	}
	Logf("Cloud Config File bytes written %d at location %s", bytesWritten, f.CloudConfigPath)
	return nil
}

func (f *Framework) SaveKubeConfig(kubeconfig string) error {
	kubeconfigFile, err := os.Create(f.ClusterKubeconfigPath)
	if err != nil {
		return err
	}
	defer kubeconfigFile.Close()
	bytesWritten, err := kubeconfigFile.WriteString(kubeconfig)
	if err != nil {
		return err
	}
	Logf("Kubeconfig File bytes written %d at location %s", bytesWritten, f.ClusterKubeconfigPath)
	return nil
}

func (f *Framework) setImages() {
	var Agnhost = "agnhost:2.6"
	var BusyBoxImage = "busybox:latest"
	var Nginx = "nginx:stable-alpine"
	var Centos = "centos:latest"

	if architecture == "ARM" {
		Agnhost = "agnhost-arm:2.6"
		BusyBoxImage = "busybox-arm:latest"
		Nginx = "nginx-arm:latest"
		Centos = "centos-arm:latest"
	}

	if imagePullRepo != "" {
		agnhost = fmt.Sprintf("%s%s", imagePullRepo, Agnhost)
		busyBoxImage = fmt.Sprintf("%s%s", imagePullRepo, BusyBoxImage)
		nginx = fmt.Sprintf("%s%s", imagePullRepo, Nginx)
		centos = fmt.Sprintf("%s%s", imagePullRepo, Centos)
	} else {
		agnhost = imageutils.GetE2EImage(imageutils.Agnhost)
		busyBoxImage = BusyBoxImage
		nginx = Nginx
		centos = Centos
	}
}

func (f *CloudProviderFramework) GetCompartmentId(setupF Framework) string {
	compartmentId := ""
	if setupF.Compartment1 != "" {
		compartmentId = setupF.Compartment1
	} else if f.CloudProviderConfig.CompartmentID != "" {
		compartmentId = f.CloudProviderConfig.CompartmentID
	} else if f.CloudProviderConfig.Auth.CompartmentID != "" {
		compartmentId = f.CloudProviderConfig.Auth.CompartmentID
	} else {
		Failf("Compartment Id undefined.")
	}
	return compartmentId
}

func getBlockProvisionerName(handle string) string {
	provisioner := "blockvolume.csi.oraclecloud.com"
	if handle != "" {
		return handle + "." + provisioner
	}
	return provisioner
}

func getFSSProvisionerName(handle string) string {
	provisioner := "fss.csi.oraclecloud.com"
	if handle != "" {
		return handle + "." + provisioner
	}
	return provisioner
}

func getLustreProvisionerName(handle string) string {
	provisioner := "lustre.csi.oraclecloud.com"
	if handle != "" {
		return handle + "." + provisioner
	}
	return provisioner
}

func (f *Framework) GetCertOcid() string {
	return f.CertOCID
}
