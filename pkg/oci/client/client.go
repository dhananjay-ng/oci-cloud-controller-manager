// Copyright (C) 2018, 2025, Oracle and/or its affiliates.
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

package client

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	providercfg "github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci/config"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/compartments"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/filestorage"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	"github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/flowcontrol"
)

// defaultSynchronousAPIContextTimeout is the time we wait for synchronous APIs
// to respond before we timeout the request
const (
	defaultSynchronousAPIContextTimeout = 10 * time.Second

	defaultSynchronousAPIPollContextTimeout = 10 * time.Minute

	Ipv6Stack = "IPv6"

	ClusterIpFamilyEnv = "CLUSTER_IP_FAMILY"

	// Service Account Token expiration in seconds
	serviceAccountTokenExpiry = 21600 // 6 Hours
)

// Interface of consumed OCI API functionality.
type Interface interface {
	Compute() ComputeInterface
	LoadBalancer(*zap.SugaredLogger, string, *OCIClientConfig) GenericLoadBalancerInterface
	Networking(*OCIClientConfig) NetworkingInterface
	BlockStorage() BlockStorageInterface
	FSS(*OCIClientConfig) FileStorageInterface
	Lustre() LustreInterface
	Identity(*OCIClientConfig) IdentityInterface
	ContainerEngine() ContainerEngineInterface
	NewWorkloadIdentityClient(logger *zap.SugaredLogger, lbType string, ociClientConfig *OCIClientConfig) Interface
	CertManager() CertificateManagerInterface
}

type OCIClientConfig struct {
	Sa           *v1.ServiceAccount
	SaToken      *authv1.TokenRequest
	ParentRptURL string
	TenancyId    string
}

// RateLimiter reader and writer.
type RateLimiter struct {
	Reader flowcontrol.RateLimiter
	Writer flowcontrol.RateLimiter
}

type computeClient interface {
	GetInstance(ctx context.Context, request core.GetInstanceRequest) (response core.GetInstanceResponse, err error)
	ListInstances(ctx context.Context, request core.ListInstancesRequest) (response core.ListInstancesResponse, err error)
	ListVnicAttachments(ctx context.Context, request core.ListVnicAttachmentsRequest) (response core.ListVnicAttachmentsResponse, err error)

	GetVnicAttachment(ctx context.Context, request core.GetVnicAttachmentRequest) (response core.GetVnicAttachmentResponse, err error)
	AttachVnic(ctx context.Context, request core.AttachVnicRequest) (response core.AttachVnicResponse, err error)

	GetVolumeAttachment(ctx context.Context, request core.GetVolumeAttachmentRequest) (response core.GetVolumeAttachmentResponse, err error)
	ListVolumeAttachments(ctx context.Context, request core.ListVolumeAttachmentsRequest) (response core.ListVolumeAttachmentsResponse, err error)
	AttachVolume(ctx context.Context, request core.AttachVolumeRequest) (response core.AttachVolumeResponse, err error)
	DetachVolume(ctx context.Context, request core.DetachVolumeRequest) (response core.DetachVolumeResponse, err error)
	ListInstanceDevices(ctx context.Context, request core.ListInstanceDevicesRequest) (response core.ListInstanceDevicesResponse, err error)
}

type virtualNetworkClient interface {
	GetVnic(ctx context.Context, request core.GetVnicRequest) (response core.GetVnicResponse, err error)
	GetSubnet(ctx context.Context, request core.GetSubnetRequest) (response core.GetSubnetResponse, err error)
	GetVcn(ctx context.Context, request core.GetVcnRequest) (response core.GetVcnResponse, err error)
	GetSecurityList(ctx context.Context, request core.GetSecurityListRequest) (response core.GetSecurityListResponse, err error)
	UpdateSecurityList(ctx context.Context, request core.UpdateSecurityListRequest) (response core.UpdateSecurityListResponse, err error)

	GetPrivateIp(ctx context.Context, request core.GetPrivateIpRequest) (response core.GetPrivateIpResponse, err error)
	ListPrivateIps(ctx context.Context, request core.ListPrivateIpsRequest) (response core.ListPrivateIpsResponse, err error)
	CreatePrivateIp(ctx context.Context, request core.CreatePrivateIpRequest) (response core.CreatePrivateIpResponse, err error)

	ListIpv6s(ctx context.Context, request core.ListIpv6sRequest) (response core.ListIpv6sResponse, err error)
	CreateIpv6(ctx context.Context, request core.CreateIpv6Request) (response core.CreateIpv6Response, err error)

	GetPublicIpByIpAddress(ctx context.Context, request core.GetPublicIpByIpAddressRequest) (response core.GetPublicIpByIpAddressResponse, err error)
	GetIpv6(ctx context.Context, request core.GetIpv6Request) (response core.GetIpv6Response, err error)

	CreateNetworkSecurityGroup(ctx context.Context, request core.CreateNetworkSecurityGroupRequest) (response core.CreateNetworkSecurityGroupResponse, err error)
	GetNetworkSecurityGroup(ctx context.Context, request core.GetNetworkSecurityGroupRequest) (response core.GetNetworkSecurityGroupResponse, err error)
	ListNetworkSecurityGroups(ctx context.Context, request core.ListNetworkSecurityGroupsRequest) (response core.ListNetworkSecurityGroupsResponse, err error)
	UpdateNetworkSecurityGroup(ctx context.Context, request core.UpdateNetworkSecurityGroupRequest) (response core.UpdateNetworkSecurityGroupResponse, err error)
	DeleteNetworkSecurityGroup(ctx context.Context, request core.DeleteNetworkSecurityGroupRequest) (response core.DeleteNetworkSecurityGroupResponse, err error)

	AddNetworkSecurityGroupSecurityRules(ctx context.Context, request core.AddNetworkSecurityGroupSecurityRulesRequest) (response core.AddNetworkSecurityGroupSecurityRulesResponse, err error)
	RemoveNetworkSecurityGroupSecurityRules(ctx context.Context, request core.RemoveNetworkSecurityGroupSecurityRulesRequest) (response core.RemoveNetworkSecurityGroupSecurityRulesResponse, err error)
	ListNetworkSecurityGroupSecurityRules(ctx context.Context, request core.ListNetworkSecurityGroupSecurityRulesRequest) (response core.ListNetworkSecurityGroupSecurityRulesResponse, err error)
	UpdateNetworkSecurityGroupSecurityRules(ctx context.Context, request core.UpdateNetworkSecurityGroupSecurityRulesRequest) (response core.UpdateNetworkSecurityGroupSecurityRulesResponse, err error)
}

type loadBalancerClient interface {
	GetLoadBalancer(ctx context.Context, request loadbalancer.GetLoadBalancerRequest) (response loadbalancer.GetLoadBalancerResponse, err error)
	ListLoadBalancers(ctx context.Context, request loadbalancer.ListLoadBalancersRequest) (response loadbalancer.ListLoadBalancersResponse, err error)
	CreateLoadBalancer(ctx context.Context, request loadbalancer.CreateLoadBalancerRequest) (response loadbalancer.CreateLoadBalancerResponse, err error)
	DeleteLoadBalancer(ctx context.Context, request loadbalancer.DeleteLoadBalancerRequest) (response loadbalancer.DeleteLoadBalancerResponse, err error)
	ListCertificates(ctx context.Context, request loadbalancer.ListCertificatesRequest) (response loadbalancer.ListCertificatesResponse, err error)
	CreateCertificate(ctx context.Context, request loadbalancer.CreateCertificateRequest) (response loadbalancer.CreateCertificateResponse, err error)
	GetWorkRequest(ctx context.Context, request loadbalancer.GetWorkRequestRequest) (response loadbalancer.GetWorkRequestResponse, err error)
	ListWorkRequests(ctx context.Context, request loadbalancer.ListWorkRequestsRequest) (response loadbalancer.ListWorkRequestsResponse, err error)
	CreateBackendSet(ctx context.Context, request loadbalancer.CreateBackendSetRequest) (response loadbalancer.CreateBackendSetResponse, err error)
	UpdateBackendSet(ctx context.Context, request loadbalancer.UpdateBackendSetRequest) (response loadbalancer.UpdateBackendSetResponse, err error)
	DeleteBackendSet(ctx context.Context, request loadbalancer.DeleteBackendSetRequest) (response loadbalancer.DeleteBackendSetResponse, err error)
	GetBackendSetHealth(ctx context.Context, request loadbalancer.GetBackendSetHealthRequest) (response loadbalancer.GetBackendSetHealthResponse, err error)
	CreateListener(ctx context.Context, request loadbalancer.CreateListenerRequest) (response loadbalancer.CreateListenerResponse, err error)
	UpdateListener(ctx context.Context, request loadbalancer.UpdateListenerRequest) (response loadbalancer.UpdateListenerResponse, err error)
	DeleteListener(ctx context.Context, request loadbalancer.DeleteListenerRequest) (response loadbalancer.DeleteListenerResponse, err error)
	CreateRuleSet(ctx context.Context, request loadbalancer.CreateRuleSetRequest) (response loadbalancer.CreateRuleSetResponse, err error)
	UpdateRuleSet(ctx context.Context, request loadbalancer.UpdateRuleSetRequest) (response loadbalancer.UpdateRuleSetResponse, err error)
	DeleteRuleSet(ctx context.Context, request loadbalancer.DeleteRuleSetRequest) (response loadbalancer.DeleteRuleSetResponse, err error)
	UpdateLoadBalancerShape(ctx context.Context, request loadbalancer.UpdateLoadBalancerShapeRequest) (response loadbalancer.UpdateLoadBalancerShapeResponse, err error)
	UpdateNetworkSecurityGroups(ctx context.Context, request loadbalancer.UpdateNetworkSecurityGroupsRequest) (response loadbalancer.UpdateNetworkSecurityGroupsResponse, err error)
	UpdateLoadBalancer(ctx context.Context, request loadbalancer.UpdateLoadBalancerRequest) (response loadbalancer.UpdateLoadBalancerResponse, err error)
}

type networkLoadBalancerClient interface {
	GetNetworkLoadBalancer(ctx context.Context, request networkloadbalancer.GetNetworkLoadBalancerRequest) (response networkloadbalancer.GetNetworkLoadBalancerResponse, err error)
	ListNetworkLoadBalancers(ctx context.Context, request networkloadbalancer.ListNetworkLoadBalancersRequest) (response networkloadbalancer.ListNetworkLoadBalancersResponse, err error)
	CreateNetworkLoadBalancer(ctx context.Context, request networkloadbalancer.CreateNetworkLoadBalancerRequest) (response networkloadbalancer.CreateNetworkLoadBalancerResponse, err error)
	DeleteNetworkLoadBalancer(ctx context.Context, request networkloadbalancer.DeleteNetworkLoadBalancerRequest) (response networkloadbalancer.DeleteNetworkLoadBalancerResponse, err error)
	GetWorkRequest(ctx context.Context, request networkloadbalancer.GetWorkRequestRequest) (response networkloadbalancer.GetWorkRequestResponse, err error)
	ListWorkRequests(ctx context.Context, request networkloadbalancer.ListWorkRequestsRequest) (response networkloadbalancer.ListWorkRequestsResponse, err error)
	CreateBackendSet(ctx context.Context, request networkloadbalancer.CreateBackendSetRequest) (response networkloadbalancer.CreateBackendSetResponse, err error)
	UpdateBackendSet(ctx context.Context, request networkloadbalancer.UpdateBackendSetRequest) (response networkloadbalancer.UpdateBackendSetResponse, err error)
	DeleteBackendSet(ctx context.Context, request networkloadbalancer.DeleteBackendSetRequest) (response networkloadbalancer.DeleteBackendSetResponse, err error)
	GetBackendSetHealth(ctx context.Context, request networkloadbalancer.GetBackendSetHealthRequest) (response networkloadbalancer.GetBackendSetHealthResponse, err error)
	CreateListener(ctx context.Context, request networkloadbalancer.CreateListenerRequest) (response networkloadbalancer.CreateListenerResponse, err error)
	UpdateListener(ctx context.Context, request networkloadbalancer.UpdateListenerRequest) (response networkloadbalancer.UpdateListenerResponse, err error)
	DeleteListener(ctx context.Context, request networkloadbalancer.DeleteListenerRequest) (response networkloadbalancer.DeleteListenerResponse, err error)
	UpdateNetworkSecurityGroups(ctx context.Context, request networkloadbalancer.UpdateNetworkSecurityGroupsRequest) (response networkloadbalancer.UpdateNetworkSecurityGroupsResponse, err error)
	UpdateNetworkLoadBalancer(ctx context.Context, request networkloadbalancer.UpdateNetworkLoadBalancerRequest) (response networkloadbalancer.UpdateNetworkLoadBalancerResponse, err error)
}

type filestorageClient interface {
	CreateFileSystem(ctx context.Context, request filestorage.CreateFileSystemRequest) (response filestorage.CreateFileSystemResponse, err error)
	GetFileSystem(ctx context.Context, request filestorage.GetFileSystemRequest) (response filestorage.GetFileSystemResponse, err error)
	ListFileSystems(ctx context.Context, request filestorage.ListFileSystemsRequest) (response filestorage.ListFileSystemsResponse, err error)
	DeleteFileSystem(ctx context.Context, request filestorage.DeleteFileSystemRequest) (response filestorage.DeleteFileSystemResponse, err error)
	UpdateFileSystem(ctx context.Context, request filestorage.UpdateFileSystemRequest) (response filestorage.UpdateFileSystemResponse, err error)

	CreateExport(ctx context.Context, request filestorage.CreateExportRequest) (response filestorage.CreateExportResponse, err error)
	ListExports(ctx context.Context, request filestorage.ListExportsRequest) (response filestorage.ListExportsResponse, err error)
	GetExport(ctx context.Context, request filestorage.GetExportRequest) (response filestorage.GetExportResponse, err error)
	DeleteExport(ctx context.Context, request filestorage.DeleteExportRequest) (response filestorage.DeleteExportResponse, err error)

	GetMountTarget(ctx context.Context, request filestorage.GetMountTargetRequest) (response filestorage.GetMountTargetResponse, err error)
	CreateMountTarget(ctx context.Context, request filestorage.CreateMountTargetRequest) (response filestorage.CreateMountTargetResponse, err error)
	DeleteMountTarget(ctx context.Context, request filestorage.DeleteMountTargetRequest) (response filestorage.DeleteMountTargetResponse, err error)
	ListMountTargets(ctx context.Context, request filestorage.ListMountTargetsRequest) (response filestorage.ListMountTargetsResponse, err error)
}

type blockstorageClient interface {
	GetVolume(ctx context.Context, request core.GetVolumeRequest) (response core.GetVolumeResponse, err error)
	CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error)
	DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error)
	ListVolumes(ctx context.Context, request core.ListVolumesRequest) (response core.ListVolumesResponse, err error)
	UpdateVolume(ctx context.Context, request core.UpdateVolumeRequest) (response core.UpdateVolumeResponse, err error)
	GetBootVolume(ctx context.Context, request core.GetBootVolumeRequest) (response core.GetBootVolumeResponse, err error)

	GetVolumeBackup(ctx context.Context, request core.GetVolumeBackupRequest) (response core.GetVolumeBackupResponse, err error)
	CreateVolumeBackup(ctx context.Context, request core.CreateVolumeBackupRequest) (response core.CreateVolumeBackupResponse, err error)
	DeleteVolumeBackup(ctx context.Context, request core.DeleteVolumeBackupRequest) (response core.DeleteVolumeBackupResponse, err error)
	ListVolumeBackups(ctx context.Context, request core.ListVolumeBackupsRequest) (response core.ListVolumeBackupsResponse, err error)
}

type identityClient interface {
	ListAvailabilityDomains(ctx context.Context, request identity.ListAvailabilityDomainsRequest) (identity.ListAvailabilityDomainsResponse, error)
}

type containerEngineClient interface {
	GetVirtualNode(ctx context.Context, request containerengine.GetVirtualNodeRequest) (response containerengine.GetVirtualNodeResponse, err error)
	RebootClusterNode(ctx context.Context, request containerengine.RebootClusterNodeRequest) (response containerengine.RebootClusterNodeResponse, err error)
	ReplaceBootVolumeClusterNode(ctx context.Context, request containerengine.ReplaceBootVolumeClusterNodeRequest) (response containerengine.ReplaceBootVolumeClusterNodeResponse, err error)
	GetWorkRequest(ctx context.Context, request containerengine.GetWorkRequestRequest) (response containerengine.GetWorkRequestResponse, err error)
	DeleteWorkRequest(ctx context.Context, request containerengine.DeleteWorkRequestRequest) (response containerengine.DeleteWorkRequestResponse, err error)
}

type compartmentClient interface {
	ListAvailabilityDomains(ctx context.Context, request compartments.ListAvailabilityDomainsRequest) (compartments.ListAvailabilityDomainsResponse, error)
}

type client struct {
	compute                      computeClient
	network                      virtualNetworkClient
	loadbalancer                 GenericLoadBalancerInterface
	networkloadbalancer          GenericLoadBalancerInterface
	filestorage                  filestorageClient
	lustre                       lustrefilestorage.LustreFileStorageClient
	bs                           blockstorageClient
	identity                     identityClient
	containerEngine              containerEngineClient
	compartment                  compartmentClient
	certificatesManagementClient certificatesmanagement.CertificatesManagementClient

	requestMetadata common.RequestMetadata
	rateLimiter     RateLimiter

	subnetCache                         cache.Store
	instanceIdToPrimaryVnicIDCache      sync.Map
	instanceIdToPrimaryVnicDetailsCache sync.Map
	configProviderCache                 cache.Store
	logger                              *zap.SugaredLogger
}

func setupBaseClient(log *zap.SugaredLogger, client *common.BaseClient, signer common.HTTPRequestSigner, interceptor common.RequestInterceptor, endpointOverrideEnvVar string) {
	client.Signer = signer
	client.Interceptor = interceptor
	if endpointOverrideEnvVar != "" {
		endpointOverride, ok := os.LookupEnv(endpointOverrideEnvVar)
		if ok && endpointOverride != "" {
			client.Host = endpointOverride
		}
	}
	clusterIpFamily, ok := os.LookupEnv(ClusterIpFamilyEnv)
	// currently as dual stack endpoints are going to be present in selected regions, only for IPv6 single stack cluster we will be using dual stack endpoints
	if ok && strings.EqualFold(clusterIpFamily, Ipv6Stack) {
		client.EnableDualStackEndpoints(true)

		region, ok := os.LookupEnv("OCI_RESOURCE_PRINCIPAL_REGION")
		if !ok {
			log.Errorf("unable to get OCI_RESOURCE_PRINCIPAL_REGION env var for region")
		}

		authEndpoint, ok := os.LookupEnv("OCI_SDK_AUTH_CLIENT_REGION_URL")
		if !ok {
			authDualStackEndpoint := common.StringToRegion(region).EndpointForTemplate("", "ds.auth.{region}.oci.{secondLevelDomain}")
			if err := os.Setenv("OCI_SDK_AUTH_CLIENT_REGION_URL", authDualStackEndpoint); err != nil {
				log.Errorf("unable to set OCI_SDK_AUTH_CLIENT_REGION_URL env var for oci auth dual stack endpoint")
			} else {
				log.Infof("OCI_SDK_AUTH_CLIENT_REGION_URL env var set to: %s", authDualStackEndpoint)
			}
		} else {
			log.Infof("OCI_SDK_AUTH_CLIENT_REGION_URL env var set to: %s", authEndpoint)
		}
	}
}

// New constructs an OCI API client.
func New(logger *zap.SugaredLogger, cp common.ConfigurationProvider, opRateLimiter *RateLimiter, cloudProviderConfig *providercfg.Config) (Interface, error) {

	signer := common.RequestSigner(cp, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
	interceptor := func(r *http.Request) error {
		r.Header.Set("x-cross-tenancy-request", cloudProviderConfig.Auth.TenancyID)
		return nil
	}

	compute, err := core.NewComputeClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewComputeClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &compute.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &compute.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring load balancer client custom transport")
	}

	network, err := core.NewVirtualNetworkClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewVirtualNetworkClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &network.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &network.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring load balancer client custom transport")
	}

	lb, err := loadbalancer.NewLoadBalancerClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewLoadBalancerClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &lb.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &lb.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring loadbalancer client custom transport")
	}

	nlb, err := networkloadbalancer.NewNetworkLoadBalancerClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewNetworkLoadBalancerClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &nlb.BaseClient, signer, interceptor, "NLB_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &nlb.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring networkloadbalancer client custom transport")
	}

	identity, err := identity.NewIdentityClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewIdentityClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &identity.BaseClient, signer, interceptor, "ID_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &identity.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring identity service client custom transport")
	}

	compartment, err := compartments.NewCompartmentsClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewCompartmentsClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &compartment.BaseClient, signer, interceptor, "COMPARTMENT_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &compartment.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring compartment service client custom transport")
	}

	bs, err := core.NewBlockstorageClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewBlockstorageClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &bs.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &bs.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring block storage service client custom transport")
	}

	fss, err := filestorage.NewFileStorageClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewFileStorageClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &fss.BaseClient, signer, interceptor, "FSS_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &fss.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring file storage service client custom transport")
	}

	// Lustre File Storage client
	lustreClient, err := lustrefilestorage.NewLustreFileStorageClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewLustreFileStorageClientWithConfigurationProvider")
	}
	setupBaseClient(logger, &lustreClient.BaseClient, signer, interceptor, "LUSTRE_ENDPOINT_OVERRIDE")
	err = configureCustomTransport(logger, &lustreClient.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring lustre file storage client custom transport")
	}

	containerEngine, err := containerengine.NewContainerEngineClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "NewContainerEngineClientWithConfigurationProvider")
	}

	setupBaseClient(logger, &containerEngine.BaseClient, signer, interceptor, "CE_ENDPOINT_OVERRIDE")

	err = configureCustomTransport(logger, &containerEngine.BaseClient)
	if err != nil {
		return nil, errors.Wrap(err, "configuring container engine service client custom transport")
	}

	requestMetadata := common.RequestMetadata{
		RetryPolicy: newRetryPolicy(),
	}

	loadbalancer := NewLBClient(lb, requestMetadata, opRateLimiter)
	networkloadbalancer := NewNLBClient(nlb, requestMetadata, opRateLimiter)
	// Create Certificate Management tclient
	certificateClient, err := certificatesmanagement.NewCertificatesManagementClientWithConfigurationProvider(cp)
	if err != nil {
		return nil, errors.Wrap(err, "configuration failed for NewCertificatesManagementClientWithConfigurationProvider")
	}

	c := &client{
		compute:                      &compute,
		network:                      &network,
		identity:                     &identity,
		loadbalancer:                 loadbalancer,
		networkloadbalancer:          networkloadbalancer,
		bs:                           &bs,
		filestorage:                  &fss,
		lustre:                       lustreClient,
		containerEngine:              &containerEngine,
		compartment:                  &compartment,
		certificatesManagementClient: certificateClient,

		rateLimiter:     *opRateLimiter,
		requestMetadata: requestMetadata,

		subnetCache:         cache.NewTTLStore(subnetCacheKeyFn, time.Duration(24)*time.Hour),
		configProviderCache: cache.NewTTLStore(providerConfigCacheKeyFn, serviceAccountTokenExpiryTime-time.Hour),
		logger:              logger,
	}

	return c, nil
}

var ServiceAccountTokenExpiry = int64(serviceAccountTokenExpiry)
var serviceAccountTokenExpiryTime = serviceAccountTokenExpiry * time.Second

type providerConfigCacheKeyValue struct {
	Key    string
	Config common.ConfigurationProvider
}

func providerConfigCacheKeyFn(obj interface{}) (string, error) {
	return obj.(providerConfigCacheKeyValue).Key, nil
}

// LoadBalancer constructs an OCI LB/NLB API client using workload identity token if service account provided
// or else returns the default cluster level client
func (c *client) LoadBalancer(logger *zap.SugaredLogger, lbType string, ociClientConfig *OCIClientConfig) (genericLoadBalancer GenericLoadBalancerInterface) {

	// tokenRequest is nil if Workload Identity LB/NLB client is not requested
	if ociClientConfig == nil || ociClientConfig.SaToken == nil {
		if lbType == "nlb" {
			return c.networkloadbalancer
		}
		if lbType == "lb" {
			return c.loadbalancer
		}
		logger.Error("Failed to get Client since load-balancer-type is neither lb or nlb!")
		return nil
	}

	// If tokenRequest is present then the requested LB/NLB client is WRIS(Workload Identity) / Nested RP based
	configProvider, err := c.getConfigurationProvider(logger, ociClientConfig)
	if err != nil {
		logger.Error("Failed to get oke workload identity RP / nested RP configuration provider! " + err.Error())
		return nil
	}

	signer := common.RequestSigner(configProvider, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
	interceptor := func(r *http.Request) error {
		r.Header.Set("x-cross-tenancy-request", ociClientConfig.TenancyId)
		return nil
	}

	if lbType == "lb" {
		lb, err := loadbalancer.NewLoadBalancerClientWithConfigurationProvider(configProvider)
		if err != nil {
			logger.Error("Failed to get new LB client with oke workload identity configuration provider! Error:" + err.Error())
			return nil
		}
		setupBaseClient(logger, &lb.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(logger, &lb.BaseClient)
		if err != nil {
			logger.Error("Failed configure custom transport for LB Client! Error:" + err.Error())
			return nil
		}

		return &loadbalancerClientStruct{
			loadbalancer:    lb,
			requestMetadata: c.requestMetadata,
			rateLimiter:     c.rateLimiter,
		}
	}
	if lbType == "nlb" {
		nlb, err := networkloadbalancer.NewNetworkLoadBalancerClientWithConfigurationProvider(configProvider)
		if err != nil {
			logger.Error("Failed to get new NLB client with oke workload identity configuration provider! Error:" + err.Error())
			return nil
		}
		setupBaseClient(logger, &nlb.BaseClient, signer, interceptor, "NLB_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(logger, &nlb.BaseClient)
		if err != nil {
			logger.Error("Failed configure custom transport for NLB Client! Error:" + err.Error())
			return nil
		}

		return &networkLoadbalancer{
			networkloadbalancer: nlb,
			requestMetadata:     c.requestMetadata,
			rateLimiter:         c.rateLimiter,
		}
	}
	logger.Error("Failed to get Client since load-balancer-type is neither lb or nlb!")
	return nil
}

func (c *client) Networking(ociClientConfig *OCIClientConfig) NetworkingInterface {
	if ociClientConfig == nil {
		return c
	}
	if ociClientConfig.SaToken != nil {
		configProvider, err := c.getConfigurationProvider(c.logger, ociClientConfig)
		if err != nil {
			c.logger.Error("Failed to get oke workload identity RP / nested RP configuration provider! " + err.Error())
			return nil
		}
		network, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
		if err != nil {
			c.logger.Errorf("Failed to create Network workload identity client %v", err)
			return nil
		}
		signer := common.RequestSigner(configProvider, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
		interceptor := func(r *http.Request) error {
			r.Header.Set("x-cross-tenancy-request", ociClientConfig.TenancyId)
			return nil
		}
		setupBaseClient(c.logger, &network.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(c.logger, &network.BaseClient)
		if err != nil {
			c.logger.Error("Failed configure custom transport for Network Client %v", err)
			return nil
		}

		return &client{
			network:         &network,
			requestMetadata: c.requestMetadata,
			rateLimiter:     c.rateLimiter,
			subnetCache:     cache.NewTTLStore(subnetCacheKeyFn, time.Duration(24)*time.Hour),
			logger:          c.logger,

			configProviderCache: c.configProviderCache,
		}
	}
	return c
}

func (c *client) Lustre() LustreInterface {
	return c
}

func (c *client) Compute() ComputeInterface {
	return c
}

func (c *client) Identity(ociClientConfig *OCIClientConfig) IdentityInterface {

	if ociClientConfig == nil {
		return c
	}
	if ociClientConfig.SaToken != nil {

		configProvider, err := c.getConfigurationProvider(c.logger, ociClientConfig)
		if err != nil {
			c.logger.Error("Failed to get oke workload identity RP / nested RP configuration provider! " + err.Error())
			return nil
		}
		identity, err := identity.NewIdentityClientWithConfigurationProvider(configProvider)
		if err != nil {
			c.logger.Errorf("Failed to create Identity workload identity  %v", err)
			return nil
		}
		signer := common.RequestSigner(configProvider, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
		interceptor := func(r *http.Request) error {
			r.Header.Set("x-cross-tenancy-request", ociClientConfig.TenancyId)
			return nil
		}
		setupBaseClient(c.logger, &identity.BaseClient, signer, interceptor, "ID_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(c.logger, &identity.BaseClient)
		if err != nil {
			c.logger.Error("Failed configure custom transport for Identity Client %v", err)
			return nil
		}

		compartment, err := compartments.NewCompartmentsClientWithConfigurationProvider(configProvider)
		if err != nil {
			c.logger.Errorf("Failed to create Compartments workload identity client  %v", err)
			return nil
		}
		setupBaseClient(c.logger, &compartment.BaseClient, signer, interceptor, "COMPARTMENT_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(c.logger, &compartment.BaseClient)
		if err != nil {
			c.logger.Error("Failed configure custom transport for Compartments Client %v", err)
			return nil
		}

		return &client{
			compartment:     &compartment,
			identity:        &identity,
			requestMetadata: c.requestMetadata,
			rateLimiter:     c.rateLimiter,
			subnetCache:     cache.NewTTLStore(subnetCacheKeyFn, time.Duration(24)*time.Hour),
			logger:          c.logger,

			configProviderCache: c.configProviderCache,
		}
	}
	return c
}

func (c *client) BlockStorage() BlockStorageInterface {
	return c
}

func (c *client) FSS(ociClientConfig *OCIClientConfig) FileStorageInterface {

	if ociClientConfig == nil {
		return c
	}
	if ociClientConfig.SaToken != nil {

		configProvider, err := c.getConfigurationProvider(c.logger, ociClientConfig)
		if err != nil {
			c.logger.Error("Failed to get oke workload identity RP / nested RP configuration provider! " + err.Error())
			return nil
		}
		fc, err := filestorage.NewFileStorageClientWithConfigurationProvider(configProvider)
		if err != nil {
			c.logger.Errorf("Failed to create FSS workload identity client %v", err)
			return nil
		}

		signer := common.RequestSigner(configProvider, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
		interceptor := func(r *http.Request) error {
			r.Header.Set("x-cross-tenancy-request", ociClientConfig.TenancyId)
			return nil
		}
		setupBaseClient(c.logger, &fc.BaseClient, signer, interceptor, "FSS_ENDPOINT_OVERRIDE")

		err = configureCustomTransport(c.logger, &fc.BaseClient)
		if err != nil {
			c.logger.Errorf("Failed configure custom transport for FSS Client %v", err.Error())
			return nil
		}

		return &client{
			filestorage:     &fc,
			requestMetadata: c.requestMetadata,
			rateLimiter:     c.rateLimiter,
			subnetCache:     cache.NewTTLStore(subnetCacheKeyFn, time.Duration(24)*time.Hour),
			logger:          c.logger,

			configProviderCache: c.configProviderCache,
		}
	}
	return c
}

func (c *client) ContainerEngine() ContainerEngineInterface {
	return c
}

func (c *client) CertManager() CertificateManagerInterface { return c }

func configureCustomTransport(logger *zap.SugaredLogger, baseClient *common.BaseClient) error {
	// no-op for internal
	return nil
}

// TODO: Reemove this once we move to having global retry policy
func getDefaultRequestMetadata(existingRequestMetadata common.RequestMetadata) common.RequestMetadata {
	if existingRequestMetadata.RetryPolicy != nil {
		return existingRequestMetadata
	}
	requestMetadata := common.RequestMetadata{
		RetryPolicy: newRetryPolicy(),
	}
	return requestMetadata
}

func (c *client) getConfigurationProvider(logger *zap.SugaredLogger, ociClientConfig *OCIClientConfig) (common.ConfigurationProvider, error) {

	// Refer cache for provider config
	configProviderCacheVal, exists, err := c.configProviderCache.GetByKey(getProviderConfigCacheKeyFromOciClientConfig(ociClientConfig))
	if exists && err == nil {
		return configProviderCacheVal.(providerConfigCacheKeyValue).Config, nil
	}

	tokenProvider := auth.NewSuppliedServiceAccountTokenProvider(ociClientConfig.SaToken.Status.Token)
	configProvider, err := auth.OkeWorkloadIdentityConfigurationProviderWithServiceAccountTokenProvider(tokenProvider)
	if err != nil {
		logger.Errorf("failed to get workload identity configuration provider %v", err.Error())
		return nil, err
	}

	if ociClientConfig.ParentRptURL != "" {
		configProvider, err = auth.ResourcePrincipalV3ConfiguratorBuilder(configProvider).WithParentRPSTURL("").WithParentRPTURL(ociClientConfig.ParentRptURL).Build()
		if err != nil {
			logger.Errorf("failed to get resource Principal configuration provider %v", err.Error())
			return nil, err
		}
	}

	// Populate provider config in cache
	c.configProviderCache.Add(providerConfigCacheKeyValue{
		Key:    getProviderConfigCacheKeyFromOciClientConfig(ociClientConfig),
		Config: configProvider,
	})
	return configProvider, nil
}

func getProviderConfigCacheKeyFromOciClientConfig(ociClientConfig *OCIClientConfig) string {
	return ociClientConfig.Sa.Namespace + string(ociClientConfig.Sa.UID) + ociClientConfig.ParentRptURL
}

func (c *client) NewWorkloadIdentityClient(logger *zap.SugaredLogger, lbType string, ociClientConfig *OCIClientConfig) Interface {

	var network core.VirtualNetworkClient

	/* In case Workload/Nested RP support is required for remaining clients
	var compute core.ComputeClient
	var identityClient identity.IdentityClient
	*/
	if ociClientConfig == nil || ociClientConfig.SaToken == nil {
		return c
	}
	configProvider, err := c.getConfigurationProvider(c.logger, ociClientConfig)
	if err != nil {
		c.logger.Error("Failed to get oke workload identity configuration provider! " + err.Error())
		return nil
	}
	signer := common.RequestSigner(configProvider, append(common.DefaultGenericHeaders(), "x-cross-tenancy-request"), common.DefaultBodyHeaders())
	interceptor := func(r *http.Request) error {
		r.Header.Set("x-cross-tenancy-request", ociClientConfig.TenancyId)
		return nil
	}

	network, err = core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		c.logger.Errorf("Failed to create Network workload identity client %v", err)
		return nil
	}
	setupBaseClient(logger, &network.BaseClient, signer, interceptor, "CORE_ENDPOINT_OVERRIDE")
	err = configureCustomTransport(c.logger, &network.BaseClient)
	if err != nil {
		c.logger.Error("Failed configure custom transport for Network Client %v", err)
		return nil
	}

	loadbalancer := c.LoadBalancer(logger, lbType, ociClientConfig)
	networkloadbalancer := loadbalancer

	return &client{
		compute:             c.compute,
		network:             network,
		identity:            c.identity,
		loadbalancer:        loadbalancer,
		networkloadbalancer: networkloadbalancer,
		bs:                  c.bs,
		filestorage:         c.filestorage,
		containerEngine:     c.containerEngine,

		rateLimiter:     c.rateLimiter,
		requestMetadata: c.requestMetadata,

		configProviderCache: c.configProviderCache,
		subnetCache:         c.subnetCache,
		logger:              logger,
	}
}
