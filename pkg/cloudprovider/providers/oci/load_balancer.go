// Copyright 2017 Oracle and/or its affiliates. All rights reserved.
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

package oci

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sports "k8s.io/kubernetes/pkg/cluster/ports"
	"k8s.io/utils/net"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/log"

	providercfg "github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci/config"
	"github.com/oracle/oci-cloud-controller-manager/pkg/metrics"
	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	"github.com/oracle/oci-cloud-controller-manager/pkg/util"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	"github.com/pkg/errors"
)

// Defines the traffic policy for load balancers created by the CCM.
const (
	DefaultLoadBalancerPolicy            = "ROUND_ROBIN"
	RoundRobinLoadBalancerPolicy         = "ROUND_ROBIN"
	LeastConnectionsLoadBalancerPolicy   = "LEAST_CONNECTIONS"
	IPHashLoadBalancerPolicy             = "IP_HASH"
	DefaultNetworkLoadBalancerPolicy     = "FIVE_TUPLE"
	NetworkLoadBalancingPolicyTwoTuple   = "TWO_TUPLE"
	NetworkLoadBalancingPolicyThreeTuple = "THREE_TUPLE"
	NetworkLoadBalancingPolicyFiveTuple  = "FIVE_TUPLE"
)

var LbOperationAlreadyExists = errors.New("An operation for the service is already in progress.")

// DefaultLoadBalancerBEProtocol defines the default protocol for load
// balancer listeners created by the CCM.
const DefaultLoadBalancerBEProtocol = "TCP"

// DefaultNetworkLoadBalancerListenerProtocol defines the default protocol for network load
// balancer listeners created by the CCM.
const DefaultNetworkLoadBalancerListenerProtocol = "TCP"

// MaxNsgPerVnic is the maximum number of NSGs that can be attached to a vnic
// https://docs.oracle.com/en-us/iaas/Content/General/Concepts/servicelimits.htm#nsg_limits
const MaxNsgPerVnic = 5

var enableOkeSystemTags = GetIsFeatureEnabledFromEnv(zap.S(), resourceTrackingFeatureFlagName, false)

const (
	// Fallback value if annotation on service is not set
	lbDefaultShape = "100Mbps"

	lbNodesHealthCheckDefaultPath             = "/healthz"
	lbNodesHealthCheckDefaultPort             = k8sports.ProxyHealthzPort
	lbNodesHealthCheckDefaultProtocol         = "HTTP"
	lbNodesHealthCheckDefaultRetries          = 3
	lbNodesHealthCheckDefaultTimeoutInMillis  = 3000
	lbNodesHealthCheckDefaultIntervalInMillis = 10000

	// default connection idle timeout per protocol
	// https://docs.cloud.oracle.com/en-us/iaas/Content/Balance/Reference/connectionreuse.htm#ConnectionConfiguration
	lbConnectionIdleTimeoutTCP       = 300
	lbConnectionIdleTimeoutHTTP      = 60
	flexible                         = "flexible"
	lbLifecycleStateActive           = "ACTIVE"
	lbMaximumNetworkSecurityGroupIds = 5
	excludeBackendFromLBLabel        = "node.kubernetes.io/exclude-from-external-load-balancers"
)

// Protects security rule addition against update by multiple LBs in parallel
var updateRulesMutex sync.Mutex

// CloudLoadBalancerProvider is an implementation of the cloud-provider struct
type CloudLoadBalancerProvider struct {
	client       client.Interface
	lbClient     client.GenericLoadBalancerInterface
	logger       *zap.SugaredLogger
	metricPusher *metrics.MetricPusher
	config       *providercfg.Config
	ociConfig    *client.OCIClientConfig
}

type IpVersions struct {
	IpFamilies               []string
	IpFamilyPolicy           *string
	LbEndpointIpVersion      *client.GenericIpVersion
	ListenerBackendIpVersion []client.GenericIpVersion
}

func (cp *CloudProvider) getLoadBalancerProvider(ctx context.Context, svc *v1.Service) (CloudLoadBalancerProvider, error) {
	lbType := getLoadBalancerType(svc)
	name := GetLoadBalancerName(svc)
	var serviceAccountToken *authv1.TokenRequest
	var serviceAccount *v1.ServiceAccount
	var opcParentRptUrl string
	var err error

	logger := cp.logger.With("loadBalancerName", name, "loadBalancerType", lbType, "serviceName", svc.Name, "serviceUid", svc.UID)
	logger.Debug("Getting load balancer provider")

	if sa, useWI := svc.Annotations[ServiceAnnotationServiceAccountName]; useWI && sa == "" { // When using Workload Identity
		return CloudLoadBalancerProvider{}, errors.New("Error fetching service account, empty string provided via " + ServiceAnnotationServiceAccountName)
	} else if useWI {
		serviceAccountToken, serviceAccount, err = cp.getServiceAccountTokenIfSet(ctx, svc)
		if err != nil {
			return CloudLoadBalancerProvider{}, errors.New("Unable to get service account token. Error:" + err.Error())
		}
		logger = logger.With("serviceAccount", sa, "nameSpace", svc.Namespace)
	}

	if s, ok := svc.Annotations[ServiceAnnotationOpcParentRptUrl]; ok {
		opcParentRptUrl = s
	}

	ociClientConfig := &client.OCIClientConfig{
		Sa:           serviceAccount,
		SaToken:      serviceAccountToken,
		TenancyId:    cp.config.Auth.TenancyID,
		ParentRptURL: opcParentRptUrl,
	}
	lbClient := cp.client.LoadBalancer(logger, lbType, ociClientConfig)
	if lbClient == nil {
		return CloudLoadBalancerProvider{}, errors.New(fmt.Sprintf("Error creating Workload Identity based %s Client. Perhaps you are using an OKE BASIC_CLUSTER?", lbType))
	}
	cpClient := cp.client.NewWorkloadIdentityClient(logger, lbType, ociClientConfig)
	if cpClient == nil {
		return CloudLoadBalancerProvider{}, errors.New(fmt.Sprintf("Error creating Workload Identity based Clients. Perhaps you are using an OKE BASIC_CLUSTER?"))
	}

	return CloudLoadBalancerProvider{
		client:       cpClient,
		lbClient:     lbClient,
		logger:       logger,
		metricPusher: cp.metricPusher,
		config:       cp.config,
		ociConfig:    ociClientConfig,
	}, nil
}

// serviceNotExistsOrDeleted returns true if service has stopped existing or has been marked as Deleted
func (cp *CloudProvider) serviceDeletedOrDoesNotExist(ctx context.Context, svc *v1.Service) (bool, error) {
	service, err := cp.kubeclient.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return true, errors.New("Unable to check if service still exists. Error:" + err.Error())
	}
	if service.DeletionTimestamp != nil {
		return true, nil
	}
	return false, nil
}

// Use Worker Identity RP based Client based on annotation: "oke.oci.oraclecloud.com/use-service-account"
// if found.
func (cp *CloudProvider) getServiceAccountTokenIfSet(ctx context.Context, svc *v1.Service) (serviceAccountTokenRequest *authv1.TokenRequest, serviceAccount *v1.ServiceAccount, err error) {
	serviceAccount, err = cp.ServiceAccountLister.ServiceAccounts(svc.Namespace).Get(svc.Annotations[ServiceAnnotationServiceAccountName])
	if err != nil {
		return nil, nil, err
	}

	serviceAccountTokenRequest = &authv1.TokenRequest{Spec: authv1.TokenRequestSpec{ExpirationSeconds: &client.ServiceAccountTokenExpiry}}

	serviceAccountTokenRequest, err = cp.kubeclient.CoreV1().ServiceAccounts(svc.Namespace).CreateToken(ctx, svc.Annotations[ServiceAnnotationServiceAccountName], serviceAccountTokenRequest, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	return
}

// GetLoadBalancerName returns the name of the loadbalancer
func (cp *CloudProvider) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			log.FromContext(ctx).
				WithValues("component", "cloud-controller-manager").
				Error(err, "Recovered from panic in GetLoadBalancerName")
			dimensionsMap := make(map[string]string)
			errorType := util.PANIC
			lbType := strings.ToUpper(getLoadBalancerType(service))
			metricDimension := util.GetComponentForMetricDimension(errorType, lbType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			dimensionsMap[metrics.ClusterOCID] = cp.config.ClusterID
			metrics.SendMetricData(cp.metricPusher, metricDimension, 1, dimensionsMap)
		}
	}()
	return GetLoadBalancerName(service)
}

// GetLoadBalancer returns whether the specified load balancer exists, and if
// so, what its status is.
func (cp *CloudProvider) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			log.FromContext(ctx).
				WithValues("component", "cloud-controller-manager").
				Error(err, "Recovered from panic in GetLoadBalancer")
			dimensionsMap := make(map[string]string)
			errorType := util.PANIC
			lbType := strings.ToUpper(getLoadBalancerType(service))
			metricDimension := util.GetComponentForMetricDimension(errorType, lbType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			dimensionsMap[metrics.ClusterOCID] = cp.config.ClusterID
			metrics.SendMetricData(cp.metricPusher, metricDimension, 1, dimensionsMap)
		}
	}()
	name := cp.GetLoadBalancerName(ctx, clusterName, service)

	lbProvider, err := cp.getLoadBalancerProvider(ctx, service)
	if err != nil {
		return nil, false, errors.Wrap(err, "Unable to get Load Balancer Client.")
	}
	logger := lbProvider.logger.With("loadBalancerName", name)
	logger.Debug("Getting load balancer")
	lb, err := lbProvider.lbClient.GetLoadBalancerByName(ctx, getLoadBalancerCompartment(service, cp.config.CompartmentID), name)
	if err != nil {
		if client.IsNotFound(err) {
			lbProvider.logger.Info("Load balancer does not exist")
			return nil, false, nil
		}

		return nil, false, err
	}
	skipPrivateIP, err := isSkipPrivateIP(service)
	if err != nil {
		return nil, false, err
	}
	lbStatus, err := loadBalancerToStatus(lb, nil, skipPrivateIP, logger)
	return lbStatus, err == nil, err
}

// getSubnets returns a list of Subnet objects for the corresponding OCIDs.
func getSubnets(ctx context.Context, subnetIDs []string, n client.NetworkingInterface) ([]*core.Subnet, error) {
	subnets := make([]*core.Subnet, len(subnetIDs))
	for i, id := range subnetIDs {
		subnet, err := n.GetSubnet(ctx, id)
		if err != nil {
			return nil, err
		}
		subnets[i] = subnet
	}
	return subnets, nil
}

// getReservedIpOcidByIpAddress returns the OCID of public reserved IP if it is in Available state.
func getReservedIpOcidByIpAddress(ctx context.Context, ipAddress string, n client.NetworkingInterface) (*string, error) {
	publicIp, err := n.GetPublicIpByIpAddress(ctx, ipAddress)
	if err != nil {
		return nil, err
	}
	if publicIp.LifecycleState != core.PublicIpLifecycleStateAvailable {
		return nil, errors.Errorf("The IP address provided is not available for use.")
	}
	return publicIp.Id, nil
}

// getSubnetsForNodes returns the de-duplicated subnets in which the given
// internal IP addresses reside.
func getSubnetsForNodes(ctx context.Context, nodes []*v1.Node, networkClient client.Interface) ([]*core.Subnet, error) {
	var (
		subnetOCIDs = sets.NewString()
		subnets     []*core.Subnet
		ipSet       = sets.New[client.IpAddresses]()
	)

	for _, node := range nodes {
		ip := NodeInternalIP(node)
		if ip.V6 == "" {
			externalIP := NodeExternalIp(node)
			if externalIP.V6 != "" {
				ip.V6 = externalIP.V6
			}
		}
		ipSet.Insert(ip)
	}

	for _, node := range nodes {
		// First see if the IP of the node belongs to a subnet in the cache.
		ip := NodeInternalIP(node)
		if ip.V6 == "" {
			// For IPv6 internal IP is not mandatory
			externalIPs := NodeExternalIp(node)
			ip.V6 = externalIPs.V6
		}

		subnet, err := networkClient.Networking(nil).GetSubnetFromCacheByIP(ip)
		if err != nil {
			return nil, err
		}
		if subnet != nil {
			// cache hit
			if !subnetOCIDs.Has(*subnet.Id) {
				subnetOCIDs.Insert(*subnet.Id)
				subnets = append(subnets, subnet)
			}
			// Since we got a cache hit we don't need to do the expensive query to find the subnet.
			continue
		}

		if node.Spec.ProviderID == "" {
			return nil, errors.Errorf(".spec.providerID was not present on node %q", node.Name)
		}

		id, err := MapProviderIDToResourceID(node.Spec.ProviderID)
		if err != nil {
			return nil, errors.Wrap(err, "MapProviderIDToResourceID")
		}

		compartmentID, ok := node.Annotations[CompartmentIDAnnotation]
		if !ok {
			return nil, errors.Errorf("%q annotation not present on node %q", CompartmentIDAnnotation, node.Name)
		}

		vnic, err := networkClient.Compute().GetPrimaryVNICForInstance(ctx, compartmentID, id)
		if err != nil {
			return nil, err
		}

		ipAddresses := client.IpAddresses{}
		if vnic != nil {
			if vnic.PrivateIp != nil {
				ipAddresses.V4 = *vnic.PrivateIp
			}
			if vnic.Ipv6Addresses != nil && len(vnic.Ipv6Addresses) > 0 {
				ipAddresses.V6 = vnic.Ipv6Addresses[0]
			}
			if ipAddresses != (client.IpAddresses{}) && ipSet.Has(ipAddresses) && !subnetOCIDs.Has(*vnic.SubnetId) {
				subnet, err := networkClient.Networking(nil).GetSubnet(ctx, *vnic.SubnetId)
				if err != nil {
					return nil, errors.Wrapf(err, "get subnet %q for instance %q", *vnic.SubnetId, id)
				}

				subnets = append(subnets, subnet)
				subnetOCIDs.Insert(*vnic.SubnetId)
			}
		}
	}
	return subnets, nil
}

// readSSLSecret returns the certificate and private key from a Kubernetes TLS
// private key Secret.
func (cp *CloudProvider) readSSLSecret(ns, name string) (*certificateData, error) {
	secret, err := cp.kubeclient.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var ok bool
	var cacert, cert, key, pass []byte
	cacert = secret.Data[SSLCAFileName]
	if cert, ok = secret.Data[SSLCertificateFileName]; !ok {
		return nil, errors.Errorf("%s not found in secret %s/%s", SSLCertificateFileName, ns, name)
	}
	if key, ok = secret.Data[SSLPrivateKeyFileName]; !ok {
		return nil, errors.Errorf("%s not found in secret %s/%s", SSLPrivateKeyFileName, ns, name)
	}
	pass = secret.Data[SSLPassphrase]
	return &certificateData{CACert: cacert, PublicCert: cert, PrivateKey: key, Passphrase: pass}, nil
}

// ensureSSLCertificate creates a OCI SSL certificate to the given load
// balancer, if it doesn't already exist.
func (clb *CloudLoadBalancerProvider) ensureSSLCertificates(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	logger := clb.logger.With("loadBalancerID", *lb.Id)
	// Get all required certificates
	certs, err := spec.Certificates()
	if err != nil {
		return err
	}

	for _, cert := range certs {
		if _, ok := lb.Certificates[*cert.CertificateName]; !ok {
			logger = logger.With("certificateName", *cert.CertificateName)
			wrID, err := clb.lbClient.CreateCertificate(ctx, *lb.Id, &cert)
			if err != nil {
				return err
			}
			logger.With("workRequestID", wrID).Info("Await workrequest for create certificate")
			_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
			if err != nil {
				return err
			}

			logger.Info("Workrequest for certificate create succeeded")
		}
	}
	return nil
}

// createLoadBalancer creates a new OCI load balancer based on the given spec.
func (clb *CloudLoadBalancerProvider) createLoadBalancer(ctx context.Context, spec *LBSpec) (lbStatus *v1.LoadBalancerStatus, lbOCID string, err error) {
	lbType := getLoadBalancerType(spec.service)
	logger := clb.logger.With("loadBalancerName", spec.Name, "loadBalancerType", lbType)
	logger.Info("Attempting to create a new load balancer")

	// First update the security lists so that if it fails (due to the etag
	// bug or otherwise) we'll retry prior to LB creation.
	lbSubnets, err := getSubnets(ctx, spec.Subnets, clb.client.Networking(nil))
	if err != nil {
		return nil, "", errors.Wrap(err, "getting subnets for load balancers")
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, spec.nodes, clb.client)
	if err != nil {
		return nil, "", errors.Wrap(err, "getting subnets for nodes")
	}

	// Then we create the load balancer and wait for it to be online.
	certs, err := spec.Certificates()
	if err != nil {
		return nil, "", errors.Wrap(err, "get certificates")
	}

	details := client.GenericCreateLoadBalancerDetails{
		CompartmentId:           &spec.Compartment,
		DisplayName:             &spec.Name,
		ShapeName:               &spec.Shape,
		IsPrivate:               &spec.Internal,
		SubnetIds:               spec.Subnets,
		BackendSets:             spec.BackendSets,
		Listeners:               spec.Listeners,
		Certificates:            certs,
		NetworkSecurityGroupIds: spec.NetworkSecurityGroupIds,
		FreeformTags:            spec.FreeformTags,
		DefinedTags:             spec.DefinedTags,
		IpVersion:               spec.IpVersions.LbEndpointIpVersion,
		RuleSets:                spec.RuleSets,
	}
	// do not block creation if the defined tag limit is reached. defer LB to tracked by backfilling
	if len(details.DefinedTags) > MaxDefinedTagPerResource {
		logger.Warnf("the number of defined tags in the LB create request is beyond the limit. removing the resource tracking tags from the details")
		delete(details.DefinedTags, OkeSystemTagNamesapce)
	}

	if _, useWI := spec.service.Annotations[ServiceAnnotationServiceAccountName]; useWI { // When using Workload Identity
		logger.Warnf("principal type is workload identity. removing oke system tags from the request")
		delete(details.DefinedTags, OkeSystemTagNamesapce)
	}

	if spec.Shape == flexible {
		details.ShapeDetails = &client.GenericShapeDetails{
			MinimumBandwidthInMbps: spec.FlexMin,
			MaximumBandwidthInMbps: spec.FlexMax,
		}
	}

	if spec.LoadBalancerIP != "" {
		reservedIpOCID, err := getReservedIpOcidByIpAddress(ctx, spec.LoadBalancerIP, clb.client.Networking(clb.ociConfig))
		if err != nil {
			return nil, "", err
		}

		details.ReservedIps = []client.GenericReservedIp{
			client.GenericReservedIp{
				Id: reservedIpOCID,
			},
		}
	}

	if lbType == NLB {
		details.CpgId = spec.ClusterPlacementGroupId
		details.AssignedPrivateIpv4 = spec.AssignedPrivateIpv4
		details.AssignedIpv6 = spec.AssignedIpv6

		if spec.IpVersionTranslationConfig != nil {
			details.IpVersionTranslationConfig = &client.IpVersionTranslationConfig{
				IpVersionTranslationMode: spec.IpVersionTranslationConfig.IpVersionTranslationMode,
				Nat46Ipv6CidrPrefix:      spec.IpVersionTranslationConfig.Nat46Ipv6CidrPrefix,
			}
		}
	}

	serviceUid := fmt.Sprintf("%s", spec.service.UID)
	wrID, err := clb.lbClient.CreateLoadBalancer(ctx, &details, &serviceUid)
	if err != nil {
		return nil, "", errors.Wrap(err, "creating load balancer")
	}
	logger.With("workRequestID", wrID).Info("Await workrequest for create loadbalancer")
	wr, err := clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return nil, "", errors.Wrap(err, "awaiting load balancer")
	}
	logger.With("workRequestID", wrID).Info("Workrequest for create loadbalancer succeeded")

	if wr.LoadBalancerId == nil {
		return nil, "", errors.New("Could not get LoadBalancerId from workrequest")
	}

	lb, err := clb.lbClient.GetLoadBalancer(ctx, *wr.LoadBalancerId)
	if err != nil {
		return nil, "", errors.Wrapf(err, "get load balancer %q", *wr.LoadBalancerId)
	}

	logger.With("loadBalancerID", *lb.Id).Info("Load balancer created")

	skipPrivateIP, err := isSkipPrivateIP(spec.service)
	if err != nil {
		return nil, "", err
	}
	status, err := loadBalancerToStatus(lb, spec.ingressIpMode, skipPrivateIP, logger)

	if status != nil && len(status.Ingress) > 0 {
		// If the LB is successfully provisioned then open lb/node subnet seclists egress/ingress.
		// Security List Updates take place in a Global Critical Section
		if err = updateSecurityListsInCriticalSection(ctx, spec, lbSubnets, nodeSubnets); err != nil {
			return nil, "", err
		}
	}
	if lb.Id != nil {
		lbOCID = *lb.Id
	}
	return status, lbOCID, err
}

// getNodeFilter extracts the node filter based on load balancer type.
// if no selector is defined then an all label selector object is returned to match everything.
func getNodeFilter(svc *v1.Service) (labels.Selector, error) {
	lbType := getLoadBalancerType(svc)

	var labelSelector string

	switch lbType {
	case NLB:
		labelSelector = svc.Annotations[ServiceAnnotationNetworkLoadBalancerNodeFilter]
	default:
		labelSelector = svc.Annotations[ServiceAnnotationLoadBalancerNodeFilter]
	}

	if labelSelector == "" {
		return labels.Everything(), nil
	}

	return labels.Parse(labelSelector)
}

// filterNodes based on the label selector, if present, and returns the set of nodes
// that should be backends in the load balancer.
func filterNodes(svc *v1.Service, nodes []*v1.Node) ([]*v1.Node, error) {

	selector, err := getNodeFilter(svc)
	if err != nil {
		return nil, err
	}

	var filteredNodes []*v1.Node
	for _, n := range nodes {
		if selector.Matches(labels.Set(n.GetLabels())) {
			filteredNodes = append(filteredNodes, n)
		}
	}

	return filteredNodes, nil
}

// filterProvisionedNodes returns a list of provisioned nodes in the provided list of nodes
func filterProvisionedNodes(nodes []*v1.Node) []*v1.Node {
	var filteredNodes []*v1.Node
	for _, n := range nodes {
		if !IsVirtualNode(n) {
			filteredNodes = append(filteredNodes, n)
		}
	}

	return filteredNodes
}

// checkSubnetIpFamilyCompatibility checks if any of the loadbalancer or node subnet supports the required IP family or returns error otherwise
func checkSubnetIpFamilyCompatibility(subnets []*core.Subnet, ipVersion string) error {
	var err error
	for _, subnet := range subnets {
		if subnet == nil {
			continue
		}
		if ipVersion == IPv6 {
			if subnet.Ipv6CidrBlock != nil || subnet.Ipv6CidrBlocks != nil {
				if len(subnet.Ipv6CidrBlocks) > 0 || *subnet.Ipv6CidrBlock != "" {
					return nil
				}
			}
			err = errors.Errorf("subnet with id %s does not have an ipv6 cidr block", pointer.StringDeref(subnet.Id, ""))
		} else {
			if subnet.CidrBlock != nil {
				// By design IPv4 CidrBlock is not allowed to be null or empty, so it has been hardcoded with string "<null>" by OCI-VCN
				if !strings.Contains(*subnet.CidrBlock, "null") {
					return nil
				}
			}
			err = errors.Errorf("subnet with id %s does not have an ipv4 cidr block", pointer.StringDeref(subnet.Id, ""))
		}

	}
	return err
}

// EnsureLoadBalancer creates a new load balancer or updates the existing one.
// Returns the status of the balancer (i.e it's public IP address if one exists).
func (cp *CloudProvider) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, clusterNodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			log.FromContext(ctx).
				WithValues("component", "cloud-controller-manager").
				Error(err, "Recovered from panic in EnsureLoadbalancer")
			dimensionsMap := make(map[string]string)
			errorType := util.PANIC
			lbType := strings.ToUpper(getLoadBalancerType(service))
			metricDimension := util.GetComponentForMetricDimension(errorType, lbType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			dimensionsMap[metrics.ClusterOCID] = cp.config.ClusterID
			metrics.SendMetricData(cp.metricPusher, metricDimension, 1, dimensionsMap)
		}
	}()
	startTime := time.Now()
	lbName := GetLoadBalancerName(service)
	loadBalancerType := getLoadBalancerType(service)

	lbProvider, err := cp.getLoadBalancerProvider(ctx, service)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get Load Balancer Client.")
	}

	logger := lbProvider.logger
	if deleted, err := cp.serviceDeletedOrDoesNotExist(ctx, service); deleted {
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to check if service exists")
			return nil, errors.Wrap(err, "Failed to check service status")
		}
		logger.Info("Service already deleted or no more exists")
		return nil, errors.New("Service already deleted or no more exists")
	}
	loadBalancerService := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if acquired := cp.lbLocks.TryAcquire(loadBalancerService); !acquired {
		logger.Error("Could not acquire lock for Ensuring Load Balancer")
		return nil, LbOperationAlreadyExists
	}
	defer cp.lbLocks.Release(loadBalancerService)

	virtualNodeExists, provisionedSvcNodes, managedPods, virtualPods, err := cp.getProvisionedNodesAndPodsOfService(ctx, logger, service, clusterNodes)
	if err != nil {
		return nil, err
	}
	// Since len(managedPods) = 0 even when it is a Flannel cluster. We cant find out if customer decided to use managed Pods on NP through this log line.
	logger.With("provisionedNodes", len(provisionedSvcNodes), "virtualPods", len(virtualPods), "managedPods", len(managedPods)).Info("Ensuring load balancer")

	// Reset node list since we need not manage resources for nodes when its Pods as Backend mode
	if isPodsAsBackendsMode(service) {
		provisionedSvcNodes = []*v1.Node{}
	}
	dimensionsMap := make(map[string]string)

	var errorType string
	var lbMetricDimension string
	var nsgMetricDimension string

	lb, err := lbProvider.lbClient.GetLoadBalancerByName(ctx, getLoadBalancerCompartment(service, cp.config.CompartmentID), lbName)
	if err != nil && !client.IsNotFound(err) {
		logger.With(zap.Error(err)).Error("Failed to get loadbalancer by name")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		dimensionsMap[metrics.ResourceOCIDDimension] = lbName
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return nil, err
	}
	lbExists := !client.IsNotFound(err)
	lbOCID := ""
	if lb != nil && lb.Id != nil {
		lbOCID = *lb.Id
	} else {
		// if the LB does not exist already use the k8s service UID for reference
		// in logs and metrics
		lbOCID = lbName
	}

	logger = lbProvider.SetLoggerWith("loadBalancerID", lbOCID)
	dimensionsMap[metrics.ResourceOCIDDimension] = lbOCID

	// Checks if we have pending work requests before processing the LoadBalancer further
	// Will error out if any in-progress work request are present for the LB
	if lb != nil && lb.Id != nil {
		err = lbProvider.checkPendingLBWorkRequests(ctx, lbProvider, lb, service, startTime)
		if err != nil {
			return nil, err
		}
	}

	var sslConfig *SSLConfig
	if requiresCertificate(service) {
		ports, err := getSSLEnabledPorts(service)
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to parse SSL port.")
			errorType = util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
			return nil, err
		}
		secretListenerString := service.Annotations[ServiceAnnotationLoadBalancerTLSSecret]
		secretBackendSetString := service.Annotations[ServiceAnnotationLoadBalancerTLSBackendSetSecret]
		sslConfig = NewSSLConfig(secretListenerString, secretBackendSetString, service, ports, cp)
		if sslConfig, err = updateSSLConfigFromCertOCID(sslConfig, service, logger); err != nil {
			logger.With(zap.Error(err)).Error("Failed to update SSL certificate.")
		}
		logger.Info("SSL Config identified %v", sslConfig)
	}

	lbSubnetIds, err := lbProvider.getLoadBalancerSubnets(ctx, service)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to get Load balancer Subnets.")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return nil, err
	}

	lbSubnets, err := getSubnets(ctx, lbSubnetIds, lbProvider.client.Networking(nil))
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get loadbalancer nodeSubnets")
		return nil, err
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, provisionedSvcNodes, cp.client)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get node nodeSubnets")
		return nil, err
	}

	ipVersions, err := lbProvider.getOciIpVersions(lbSubnets, nodeSubnets, service)
	if err != nil {
		return nil, err
	}

	ipAddressToOcidMap, err := cp.getIpAddressOcidMap(ctx, provisionedSvcNodes)
	if err != nil {
		return nil, err
	}
	spec, err := NewLBSpec(logger, service, provisionedSvcNodes, managedPods, virtualPods, lbSubnetIds, sslConfig, cp.securityListManagerFactory, ipVersions, cp.config.Tags, lb, cp.config.CompartmentID, ipAddressToOcidMap)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to derive LBSpec")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)

		return nil, err
	}

	if requiresNsgManagement(service) {
		// Fetch existing frontend NSG and use it to manage rules
		frontendNsgId := ""
		backendNsgs := spec.ManagedNetworkSecurityGroup.backendNsgId

		// Check if there are any NSGs which are created by CCM (and use that), but didn't get attached to LB because the LB creation failed.
		if !lbExists {
			frontendNsgId, _, err = lbProvider.getFrontendNsgByName(ctx, generateNsgName(service), spec.Compartment, cp.config.VCNID, fmt.Sprintf("%s", service.UID))
			if err != nil {
				return nil, err
			}
			logger.Infof("found managed NSG %s", frontendNsgId)
			if frontendNsgId != "" {
				spec, err = addFrontendNsgToSpec(spec, frontendNsgId)
				if err != nil {
					return nil, err
				}
			}
		}
		if lb != nil && lb.Id != nil && lb.NetworkSecurityGroupIds != nil {
			nsgs := lb.NetworkSecurityGroupIds
			for _, id := range nsgs {
				frontendNsgId, _, err = lbProvider.getFrontendNsg(ctx, id, fmt.Sprintf("%s", service.UID))
				if err != nil {
					errorType = util.GetError(err)
					nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
					dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
					metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Get), time.Since(startTime).Seconds(), dimensionsMap)
				}
				if frontendNsgId != "" {
					spec, err = addFrontendNsgToSpec(spec, frontendNsgId)
					if err != nil {
						return nil, err
					}
					logger.With("loadBalancerID", *lb.Id).Infof("using existing frontendNsg %s", frontendNsgId)
					break
				}
			}

			if frontendNsgId == "" {
				// Check if there are any CCM created NSGs which might be manually removed by customer causing a dirty LB
				logger.Info("Check if managed NSGs present in VCN")
				frontendNsgId, _, err = lbProvider.getFrontendNsgByName(ctx, generateNsgName(service), spec.Compartment, cp.config.VCNID, fmt.Sprintf("%s", service.UID))
				if err != nil {
					return nil, err
				}
				logger.Infof("found managed NSG %s", frontendNsgId)
				if frontendNsgId != "" {
					spec, err = addFrontendNsgToSpec(spec, frontendNsgId)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		// Create the NSG and add it to the LbSpec
		if frontendNsgId == "" {
			if len(spec.NetworkSecurityGroupIds) >= MaxNsgPerVnic {
				return nil, fmt.Errorf("invalid number of Network Security Groups (Max: 5) including managed nsg")
			}
			resp, err := lbProvider.client.Networking(nil).CreateNetworkSecurityGroup(ctx, spec.Compartment, cp.config.VCNID, generateNsgName(service), fmt.Sprintf("%s", service.UID))
			if err != nil {
				logger.With(zap.Error(err)).Error("Failed to create nsg")
				errorType = util.GetError(err)
				nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
				dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
				metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Create), time.Since(startTime).Seconds(), dimensionsMap)
				return nil, err
			}
			frontendNsgId = *resp.Id
			spec, err = addFrontendNsgToSpec(spec, frontendNsgId)
			if err != nil {
				return nil, err
			}
			logger.With("frontendNsgId", *resp.Id).
				Info("Successfully created nsg")
			nsgMetricDimension = util.GetComponentForMetricDimension(util.Success, util.NSGType)
			dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
			dimensionsMap[metrics.ResourceOCIDDimension] = *resp.Id
			metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Create), time.Since(startTime).Seconds(), dimensionsMap)
		}
		if len(backendNsgs) > 0 {
			for _, nsg := range backendNsgs {
				resp, etag, err := lbProvider.client.Networking(nil).GetNetworkSecurityGroup(ctx, nsg)
				if err != nil {
					logger.With(zap.Error(err)).Error("Failed to get nsg")
					errorType = util.GetError(err)
					nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
					dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
					metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Get), time.Since(startTime).Seconds(), dimensionsMap)
					return nil, err
				}
				freeformTags := resp.FreeformTags
				if _, ok := freeformTags[client.ManagedBy]; !ok {
					if etag != nil {
						freeformTags[client.ManagedBy] = client.CCM
						response, err := lbProvider.client.Networking(nil).UpdateNetworkSecurityGroup(ctx, nsg, *etag, freeformTags)
						if err != nil {
							logger.With(zap.Error(err)).Errorf("Failed to update nsg %s", nsg)
							errorType = util.GetError(err)
							nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
							dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
							dimensionsMap[metrics.ResourceOCIDDimension] = nsg
							metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Update), time.Since(startTime).Seconds(), dimensionsMap)
							return nil, err
						}
						nsgMetricDimension = util.GetComponentForMetricDimension(util.Success, util.NSGType)
						dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
						dimensionsMap[metrics.ResourceOCIDDimension] = *response.Id
						metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Update), time.Since(startTime).Seconds(), dimensionsMap)
					}
				}
			}
		}
		serviceComponents := securityRuleComponents{
			frontendNsgOcid:  frontendNsgId,
			backendNsgOcids:  backendNsgs,
			ports:            spec.Ports,
			sourceCIDRs:      spec.SourceCIDRs,
			isPreserveSource: *spec.IsPreserveSource,
			serviceUid:       fmt.Sprintf("service-uid-%s", service.UID),
		}
		logger.Infof("(requiresNSGmanagement) Service Components %#v", serviceComponents)
		if err = lbProvider.reconcileSecurityGroup(ctx, serviceComponents); err != nil {
			return nil, err
		}
	}

	if !lbExists {
		logger.With(ctx).Info("Attempting to create LB with specs %v", spec)
		lbStatus, newLBOCID, err := lbProvider.createLoadBalancer(ctx, spec)
		if err != nil && client.IsSystemTagNotFoundOrNotAuthorisedError(logger, err) {
			logger.With(zap.Error(err)).Warn("LB creation failed due to error in adding system tags. sending metric & retrying without system tags")

			// send resource track tagging failure metrics
			errorType = util.SystemTagErrTypePrefix + util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Create), time.Since(startTime).Seconds(), dimensionsMap)

			// retry create without resource tracking system tags
			delete(spec.DefinedTags, OkeSystemTagNamesapce)
			lbStatus, newLBOCID, err = lbProvider.createLoadBalancer(ctx, spec)
		}
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to provision LoadBalancer")
			errorType = util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Create), time.Since(startTime).Seconds(), dimensionsMap)

		} else {
			logger.With("loadBalancerID", newLBOCID).
				Info("Successfully provisioned loadbalancer")
			lbMetricDimension = util.GetComponentForMetricDimension(util.Success, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			dimensionsMap[metrics.ResourceOCIDDimension] = newLBOCID
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Create), time.Since(startTime).Seconds(), dimensionsMap)
		}

		return lbStatus, err
	}

	if lb.LifecycleState == nil || *lb.LifecycleState != lbLifecycleStateActive {
		logger := logger.With("lifecycleState", lb.LifecycleState)
		switch loadBalancerType {
		case NLB:
			// This check is added here since NLBs are marked as failed in case nlb work-requests fail NLB-26239
			if *lb.LifecycleState == string(networkloadbalancer.LifecycleStateFailed) {
				logger.Infof("NLB is in %s state, process the Loadbalancer", *lb.LifecycleState)
			} else {
				return nil, errors.Errorf("NLB is in %s state, wait for NLB to move to %s", *lb.LifecycleState, lbLifecycleStateActive)
			}
			break
		default:
			logger.Infof("LB is not in %s state, will retry EnsureLoadBalancer", lbLifecycleStateActive)
			return nil, errors.Errorf("rejecting request to update LB which is not in %s state", lbLifecycleStateActive)
		}
	}

	// Existing load balancers cannot change subnets. This ensures that the spec matches
	// what the actual load balancer has listed as the subnet ids. If the load balancer
	// was just created then these values would be equal; however, if the load balancer
	// already existed and the default subnet ids changed, then this would ensure
	// we are setting the security rules on the correct subnets.
	spec, err = updateSpecWithLbSubnets(spec, lb.SubnetIds)
	if err != nil {
		return nil, err
	}

	// If the load balancer needs an SSL cert ensure it is present.
	if requiresCertificate(service) {
		if err := lbProvider.ensureSSLCertificates(ctx, lb, spec); err != nil {
			logger.With(zap.Error(err)).Error("Failed to ensure ssl certificates")
			errorType = util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)

			return nil, errors.Wrap(err, "ensuring ssl certificates")
		}
	}

	// If network partition, do not proceed
	isNetworkPartition, err := cp.checkForNetworkPartition(logger, clusterNodes, virtualNodeExists)
	if err != nil {
		return nil, err
	} else if isNetworkPartition {
		return nil, nil
	}

	if err := lbProvider.updateLoadBalancer(ctx, lb, spec); err != nil {
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		logger.With(zap.Error(err)).Error("Failed to update LoadBalancer")
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return nil, err
	}

	syncTime := time.Since(startTime).Seconds()
	logger.Info("Successfully updated loadbalancer")
	lbMetricDimension = util.GetComponentForMetricDimension(util.Success, util.LoadBalancerType)
	dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
	dimensionsMap[metrics.BackendSetsCountDimension] = strconv.Itoa(len(lb.BackendSets))
	metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), syncTime, dimensionsMap)

	skipPrivateIP, err := isSkipPrivateIP(service)
	if err != nil {
		return nil, err
	}
	return loadBalancerToStatus(lb, spec.ingressIpMode, skipPrivateIP, logger)
}

func getDefaultLBSubnets(subnet1, subnet2 string) []string {
	var subnets []string
	if subnet2 != "" {
		subnets = []string{subnet1, subnet2}
	} else {
		subnets = []string{subnet1}
	}
	return subnets
}

func (clb *CloudLoadBalancerProvider) getNetworkLoadbalancerSubnets(ctx context.Context, svc *v1.Service) ([]string, error) {
	subnets := getDefaultLBSubnets(clb.config.LoadBalancer.Subnet1, clb.config.LoadBalancer.Subnet2)
	if s, ok := svc.Annotations[ServiceAnnotationNetworkLoadBalancerSubnet]; ok && len(s) != 0 {
		return []string{s}, nil
	}
	if len(subnets) == 0 {
		return nil, errors.Errorf("a subnet must be specified for a network load balancer to get created")
	}
	if len(subnets) > 0 && subnets[0] != "" {
		return []string{subnets[0]}, nil
	}
	return nil, errors.Errorf("a subnet must be specified for a network load balancer")
}

func (clb *CloudLoadBalancerProvider) getOciLoadBalancerSubnets(ctx context.Context, svc *v1.Service) ([]string, error) {
	internal, err := isInternalLB(svc)
	if err != nil {
		return nil, err
	}

	// NOTE: These will be overridden for existing load balancers as load
	// balancer subnets cannot be modified.
	subnets := getDefaultLBSubnets(clb.config.LoadBalancer.Subnet1, clb.config.LoadBalancer.Subnet2)

	if s, ok := svc.Annotations[ServiceAnnotationLoadBalancerSubnet1]; ok && len(s) != 0 {
		subnets[0] = s
		r, err := clb.client.Networking(nil).IsRegionalSubnet(ctx, s)
		if err != nil {
			return nil, err
		}
		if r {
			return subnets[:1], nil
		}
	}

	if s, ok := svc.Annotations[ServiceAnnotationLoadBalancerSubnet2]; ok && len(s) != 0 {
		r, err := clb.client.Networking(nil).IsRegionalSubnet(ctx, s)
		if err != nil {
			return nil, err
		}
		if r {
			subnets[0] = s
			clb.logger.Debugf("Considering annotation %s: %s for LB as it is the only regional subnet in annotations provided.", ServiceAnnotationLoadBalancerSubnet2, s)
			return subnets[:1], nil
		} else if len(subnets) > 1 {
			subnets[1] = s
		} else {
			subnets = append(subnets, s)
		}
	}

	if subnets[0] == "" || (len(subnets) == 2 && subnets[1] == "") {
		return nil, errors.Errorf("a subnet must be specified for creating a load balancer")
	}
	if internal {
		// Public load balancers need two subnets if they are AD specific and only first subnet is used if regional. Internal load
		// balancers will always use the first subnet.
		return subnets[:1], nil
	}

	return subnets, nil
}

func (clb *CloudLoadBalancerProvider) getLoadBalancerSubnets(ctx context.Context, svc *v1.Service) ([]string, error) {
	lbType := getLoadBalancerType(svc)

	switch lbType {
	case NLB:
		return clb.getNetworkLoadbalancerSubnets(ctx, svc)
	default:
		return clb.getOciLoadBalancerSubnets(ctx, svc)
	}
}

func (clb *CloudLoadBalancerProvider) updateLoadBalancer(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	lbID := *lb.Id
	start := time.Now()
	logger := clb.logger.With("loadBalancerID", lbID, "compartmentID", spec.Compartment, "loadBalancerType", getLoadBalancerType(spec.service), "serviceName", spec.service.Name)

	var actualPublicReservedIP *string

	//identify the public reserved IP in IP addresses list
	for _, ip := range lb.IpAddresses {
		if ip.IpAddress == nil {
			continue // should never happen but appears to when EnsureLoadBalancer is called with 0 nodes.
		}
		if ip.ReservedIp != nil && *ip.IsPublic {
			actualPublicReservedIP = ip.IpAddress
			break
		}
	}
	lbSubnets, err := getSubnets(ctx, spec.Subnets, clb.client.Networking(nil))
	if err != nil {
		return errors.Wrapf(err, "getting load balancer subnets")
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, spec.nodes, clb.client)
	if err != nil {
		return errors.Wrap(err, "get subnets for nodes")
	}

	// Conversion from SingleStack to DualStack needs to happen before the IPv6 listeners & Backendsets are created
	if spec.Type == NLB && spec.IpVersions.LbEndpointIpVersion != nil {
		ipVersion := string(*lb.IpVersion)
		lbEndpointVersion := string(*spec.IpVersions.LbEndpointIpVersion)
		ipFamilyPolicy := *spec.IpVersions.IpFamilyPolicy
		ipVersionChanged := hasIpVersionChanged(ipVersion, lbEndpointVersion)
		if ipVersionChanged && (ipFamilyPolicy == string(v1.IPFamilyPolicyPreferDualStack) || ipFamilyPolicy == string(v1.IPFamilyPolicyRequireDualStack)) {
			logger.Infof("IPversion: %s LbEndpointIpVersion: %s IpFamilyPolicy: %s", ipVersion, lbEndpointVersion, ipFamilyPolicy)
			details := &client.GenericUpdateLoadBalancerDetails{
				IpVersion: spec.IpVersions.LbEndpointIpVersion,
			}
			err = clb.updateLoadBalancerIpVersion(ctx, lb, details)
			if err != nil {
				return err
			}
		}
	}

	// Enablement of NAT46 translation mode can be done to Dual stack NLB
	if shouldUpdateIpVersionTranslation(lb, spec, NAT46) {
		logger.Info("IPVersionTranslation needs to be NAT46")
		details := &client.GenericUpdateLoadBalancerDetails{
			IpVersionTranslationConfig: &client.IpVersionTranslationConfig{
				IpVersionTranslationMode: spec.IpVersionTranslationConfig.IpVersionTranslationMode,
				Nat46Ipv6CidrPrefix:      spec.IpVersionTranslationConfig.Nat46Ipv6CidrPrefix,
			},
		}
		err = clb.updateLoadbalancerIpVersionTranslation(ctx, lb, details)
	}

	var ruleSetActions []Action
	if spec.RuleSets != nil {
		ruleSetActions = getRuleSetChanges(lb.RuleSets, spec.RuleSets)
	}

	actualBackendSets := lb.BackendSets
	desiredBackendSets := spec.BackendSets
	backendSetActions := getBackendSetChanges(logger, actualBackendSets, desiredBackendSets)

	actualListeners := lb.Listeners
	desiredListeners := spec.Listeners
	listenerActions := getListenerChanges(logger, actualListeners, desiredListeners, spec.RuleSets)

	if len(backendSetActions) == 0 && len(listenerActions) == 0 {
		// If there are no backendSetActions or Listener actions
		// this function must have been called because of a failed
		// seclist update when the load balancer was created
		// We try to update the seclist this way to prevent replication
		// of seclist reconciliation logic
		// Security List Updates happen in a Global Critical Section
		if err = updateSecurityListsInCriticalSection(ctx, spec, lbSubnets, nodeSubnets); err != nil {
			return err
		}
	}
	actions := sortAndCombineActions(logger, backendSetActions, listenerActions, ruleSetActions)

	for _, action := range actions {
		switch a := action.(type) {
		case *BackendSetAction:
			err := clb.updateBackendSet(ctx, lbID, a, lbSubnets, nodeSubnets, spec.securityListManager, spec)
			if err != nil {
				return errors.Wrap(err, "updating BackendSet")
			}
		case *ListenerAction:
			backendSetName := *a.Listener.DefaultBackendSetName
			var ports portSpec
			if a.Type() == Delete {
				// If we need to delete the BackendSet then it'll no longer be
				// present in the spec since that's what is desired, so we need
				// to fetch it from the load balancer object.
				bs := lb.BackendSets[backendSetName]
				ports = portsFromBackendSet(logger, backendSetName, &bs)
			} else {
				ports = spec.Ports[backendSetName]
			}

			err := clb.updateListener(ctx, lbID, a, ports, lbSubnets, nodeSubnets, spec.SourceCIDRs, spec.securityListManager, spec)
			if err != nil {
				return errors.Wrap(err, "updating listener")
			}
		case *RuleSetAction:
			err := clb.updateRuleSet(ctx, lbID, a, spec)
			if err != nil {
				return errors.Wrap(err, "updating RuleSet")
			}
		}
	}

	// Check if the customer managed LB NSGs have changed
	nsgChanged := hasLoadBalancerNetworkSecurityGroupsChanged(ctx, lb.NetworkSecurityGroupIds, spec.NetworkSecurityGroupIds)
	if nsgChanged {
		err = clb.updateLoadBalancerNetworkSecurityGroups(ctx, lb, spec)
		if err != nil {
			return err
		}
	}

	// Only LB supports fixed/flexible shapes which can be changed
	if spec.Type == LB {
		shapeChanged := hasLoadbalancerShapeChanged(ctx, spec, lb)

		if shapeChanged {
			err = clb.updateLoadbalancerShape(ctx, lb, spec)
			if err != nil {
				return err
			}
		}
	}

	// IP Version Translation is set to DISABLED after reconciling Listener and BackendSet, irrelevant of IPFamily
	// IP Version should be DISABLED after Listener Backend actions
	if shouldUpdateIpVersionTranslation(lb, spec, DISABLED) {
		logger.Info("IPVersionTranslation needs to be disabled")
		details := &client.GenericUpdateLoadBalancerDetails{
			IpVersionTranslationConfig: &client.IpVersionTranslationConfig{
				IpVersionTranslationMode: spec.IpVersionTranslationConfig.IpVersionTranslationMode,
				Nat46Ipv6CidrPrefix:      spec.IpVersionTranslationConfig.Nat46Ipv6CidrPrefix,
			},
		}
		err = clb.updateLoadbalancerIpVersionTranslation(ctx, lb, details)
	}

	// Conversion from DualStack to SingleStack needs to happen after the IPv6 listeners & Backendsets are removed
	if spec.Type == NLB && spec.IpVersions.LbEndpointIpVersion != nil {
		ipVersion := string(*lb.IpVersion)
		lbEndpointVersion := string(*spec.IpVersions.LbEndpointIpVersion)
		ipFamilyPolicy := *spec.IpVersions.IpFamilyPolicy
		ipVersionChanged := hasIpVersionChanged(ipVersion, lbEndpointVersion)
		if ipVersionChanged && ipFamilyPolicy == string(v1.IPFamilyPolicySingleStack) {
			logger.Infof("IPversion: %s LbEndpointIpVersion: %s IpFamilyPolicy: %s", ipVersion, lbEndpointVersion, ipFamilyPolicy)
			details := &client.GenericUpdateLoadBalancerDetails{
				IpVersion: spec.IpVersions.LbEndpointIpVersion,
			}
			err = clb.updateLoadBalancerIpVersion(ctx, lb, details)
			if err != nil {
				return err
			}
		}
	}

	// Check if the reservedIP has changed in spec
	if spec.LoadBalancerIP != "" || actualPublicReservedIP != nil {
		if actualPublicReservedIP == nil || *actualPublicReservedIP != spec.LoadBalancerIP {
			return errors.Errorf("The Load Balancer service reserved IP cannot be updated after the Load Balancer is created.")
		}
	}

	dimensionsMap := make(map[string]string)
	var errType string
	if enableOkeSystemTags && !doesLbHaveOkeSystemTags(lb, spec) {
		logger.Info("detected loadbalancer without oke system tags. proceeding to add")
		err = clb.addLoadBalancerOkeSystemTags(ctx, lb, spec)
		if err != nil {
			// fail open if the update request fails
			logger.With(zap.Error(err)).Warn("updateLoadBalancer didn't succeed. unable to add oke system tags")
			errType = util.SystemTagErrTypePrefix + util.GetError(err)
			if errors.Is(err, fmt.Errorf(MaxDefinedTagErrMessage, spec.Type)) {
				errType = util.ErrTagLimitReached
			}
			dimensionsMap[metrics.ComponentDimension] = util.GetComponentForMetricDimension(errType, util.LoadBalancerType)
			dimensionsMap[metrics.ResourceOCIDDimension] = *lb.Id
			metrics.SendMetricData(clb.metricPusher, getMetric(spec.Type, Update), time.Since(start).Seconds(), dimensionsMap)
		}
	}
	return nil
}

func (clb *CloudLoadBalancerProvider) updateLoadBalancerBackends(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	lbID := *lb.Id

	logger := clb.logger.With("loadBalancerID", lbID, "compartmentID", spec.Compartment, "loadBalancerType", getLoadBalancerType(spec.service), "serviceName", spec.service.Name)

	lbSubnets, err := getSubnets(ctx, spec.Subnets, clb.client.Networking(nil))
	if err != nil {
		return errors.Wrapf(err, "getting load balancer subnets")
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, spec.nodes, clb.client)
	if err != nil {
		return errors.Wrap(err, "get subnets for nodes")
	}

	actualBackendSets := lb.BackendSets
	desiredBackendSets := spec.BackendSets
	backendSetActions := getBackendSetChanges(logger, actualBackendSets, desiredBackendSets)

	for _, action := range backendSetActions {
		switch a := action.(type) {
		case *BackendSetAction:
			switch a.Type() {
			case Update:
				err := clb.updateBackendSet(ctx, lbID, a, lbSubnets, nodeSubnets, spec.securityListManager, spec)
				if err != nil {
					return errors.Wrap(err, "updating BackendSet")
				}
			}
		}
	}
	return nil
}

func updateSecurityListsInCriticalSection(ctx context.Context, spec *LBSpec, lbSubnets, nodeSubnets []*core.Subnet) (err error) {
	updateRulesMutex.Lock()
	defer updateRulesMutex.Unlock()
	for _, ports := range spec.Ports {
		sc := securityRuleComponents{
			lbSubnets:        lbSubnets,
			backendSubnets:   nodeSubnets,
			sourceCIDRs:      spec.SourceCIDRs,
			actualPorts:      nil,
			desiredPorts:     ports,
			isPreserveSource: *spec.IsPreserveSource,
			ipFamilies:       convertOciIpVersionsToOciIpFamilies(spec.IpVersions.ListenerBackendIpVersion),
		}
		if err = spec.securityListManager.Update(ctx, sc); err != nil {
			return err
		}
	}
	return nil
}

func (clb *CloudLoadBalancerProvider) updateBackendSet(ctx context.Context, lbID string, action *BackendSetAction, lbSubnets, nodeSubnets []*core.Subnet, secListManager securityListManager, spec *LBSpec) error {
	var (
		sourceCIDRs   = []string{}
		workRequestID string
		err           error
		bs            = action.BackendSet
		ports         = action.Ports
	)

	logger := clb.logger.With(
		"actionType", action.Type(),
		"backendSetName", action.Name(),
		"ports", ports,
		"loadBalancerID", lbID,
		"loadBalancerType", getLoadBalancerType(spec.service))
	logger.Info("Applying action on backend set")

	sc := securityRuleComponents{
		lbSubnets:        lbSubnets,
		backendSubnets:   nodeSubnets,
		sourceCIDRs:      sourceCIDRs,
		actualPorts:      nil,
		desiredPorts:     ports,
		isPreserveSource: *spec.IsPreserveSource,
		ipFamilies:       convertOciIpVersionsToOciIpFamilies(spec.IpVersions.ListenerBackendIpVersion),
	}

	switch action.Type() {
	case Create:
		err = secListManager.Update(ctx, sc)
		if err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.CreateBackendSet(ctx, lbID, action.Name(), &bs)
	case Update:
		// For NLB, due to source IP preservation we need to ensure ingress rules from sourceCIDRs are added to
		// the backends subnet's seclist as well
		sc.actualPorts = action.OldPorts
		sc.sourceCIDRs = spec.SourceCIDRs
		if err = secListManager.Update(ctx, sc); err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.UpdateBackendSet(ctx, lbID, action.Name(), &bs)
	case Delete:
		err = secListManager.Delete(ctx, sc)
		if err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.DeleteBackendSet(ctx, lbID, action.Name())
	}

	if err != nil {
		return err
	}
	logger = clb.SetLoggerWith("workRequestID", workRequestID)
	logger.Info("Await workrequest for loadbalancer backendset")
	_, err = clb.lbClient.AwaitWorkRequest(ctx, workRequestID)
	if err != nil {
		return err
	}
	logger.Info("Workrequest for loadbalancer backendset completed successfully")

	return nil
}

func (clb *CloudLoadBalancerProvider) updateListener(ctx context.Context, lbID string, action *ListenerAction, ports portSpec, lbSubnets, nodeSubnets []*core.Subnet, sourceCIDRs []string, secListManager securityListManager, spec *LBSpec) error {
	var workRequestID string
	var err error
	listener := action.Listener
	ports.ListenerPort = *listener.Port

	logger := clb.logger.With(
		"actionType", action.Type(),
		"listenerName", action.Name(),
		"ports", ports,
		"loadBalancerID", lbID,
		"loadBalancerType", getLoadBalancerType(spec.service))
	logger.Info("Applying action on listener")
	sc := securityRuleComponents{
		lbSubnets:        lbSubnets,
		backendSubnets:   nodeSubnets,
		sourceCIDRs:      sourceCIDRs,
		actualPorts:      nil,
		desiredPorts:     ports,
		isPreserveSource: *spec.IsPreserveSource,
		ipFamilies:       convertOciIpVersionsToOciIpFamilies(spec.IpVersions.ListenerBackendIpVersion),
	}
	switch action.Type() {
	case Create:
		err = secListManager.Update(ctx, sc)
		if err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.CreateListener(ctx, lbID, action.Name(), &listener)
	case Update:
		err = secListManager.Update(ctx, sc)
		if err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.UpdateListener(ctx, lbID, action.Name(), &listener)
	case Delete:
		err = secListManager.Delete(ctx, sc)
		if err != nil {
			return err
		}
		workRequestID, err = clb.lbClient.DeleteListener(ctx, lbID, action.Name())
	}

	if err != nil {
		return err
	}
	logger = clb.SetLoggerWith("workRequestID", workRequestID)
	logger.Info("Await workrequest for loadbalancer listener")
	_, err = clb.lbClient.AwaitWorkRequest(ctx, workRequestID)
	if err != nil {
		return err
	}
	logger.Info("Workrequest for loadbalancer listener completed successfully")
	return nil
}

func (clb *CloudLoadBalancerProvider) updateRuleSet(ctx context.Context, lbID string, action *RuleSetAction, spec *LBSpec) (err error) {
	var workRequestID string

	logger := clb.logger.With(
		"actionType", action.Type(),
		"ruleSetName", action.Name(),
		"loadBalancerID", lbID,
		"loadBalancerType", getLoadBalancerType(spec.service))
	logger.Info("Applying action on rule set")

	switch action.Type() {
	case Create:
		workRequestID, err = clb.lbClient.CreateRuleSet(ctx, lbID, action.Name(), &action.RuleSetDetails)
	case Update:
		workRequestID, err = clb.lbClient.UpdateRuleSet(ctx, lbID, action.Name(), &action.RuleSetDetails)
	case Delete:
		workRequestID, err = clb.lbClient.DeleteRuleSet(ctx, lbID, action.Name())
	}

	if err != nil {
		return err
	}
	logger = logger.With("workRequestID", workRequestID)
	logger.Info("Await work request for loadbalancer rule set")
	_, err = clb.lbClient.AwaitWorkRequest(ctx, workRequestID)
	if err != nil {
		return err
	}
	logger.Info("Work request for loadbalancer rule set completed successfully")

	return nil
}

// UpdateLoadBalancer updates an existing loadbalancer
func (cp *CloudProvider) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			log.FromContext(ctx).
				WithValues("component", "cloud-controller-manager").
				Error(err, "Recovered from panic in UpdateLoadBalancer")
			dimensionsMap := make(map[string]string)
			errorType := util.PANIC
			lbType := strings.ToUpper(getLoadBalancerType(service))
			metricDimension := util.GetComponentForMetricDimension(errorType, lbType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			dimensionsMap[metrics.ClusterOCID] = cp.config.ClusterID
			metrics.SendMetricData(cp.metricPusher, metricDimension, 1, dimensionsMap)
		}
	}()
	startTime := time.Now()
	lbName := GetLoadBalancerName(service)
	loadBalancerType := getLoadBalancerType(service)

	lbProvider, err := cp.getLoadBalancerProvider(ctx, service)
	if err != nil {
		return errors.Wrap(err, "Unable to get Load Balancer Client.")
	}

	logger := lbProvider.logger
	if deleted, err := cp.serviceDeletedOrDoesNotExist(ctx, service); deleted {
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to check if service exists")
			return errors.Wrap(err, "Failed to check service status")
		}
		logger.Info("Service already deleted or no more exists")
		return errors.New("Service already deleted or no more exists")
	}
	loadBalancerService := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if acquired := cp.lbLocks.TryAcquire(loadBalancerService); !acquired {
		logger.Error("Could not acquire lock for Updating Load Balancer")
		return LbOperationAlreadyExists
	}
	defer cp.lbLocks.Release(loadBalancerService)

	virtualNodeExists, provisionedSvcNodes, managedPods, virtualPods, err := cp.getProvisionedNodesAndPodsOfService(ctx, logger, service, nodes)
	if err != nil {
		return err
	}
	ipAddressToOcidMap, err := cp.getIpAddressOcidMap(ctx, provisionedSvcNodes)
	if err != nil {
		return err
	}
	// Since len(managedPods) = 0 even when it is a Flannel cluster. We cant find out if customer decided to use managed Pods on NP through this log line.
	logger.With("provisionedNodes", len(provisionedSvcNodes), "virtualPods", len(virtualPods), "managedPods", len(managedPods)).Info("Updating load balancer backends")

	// Reset node list since we need not manage resources for nodes when its Pods as Backend mode
	if isPodsAsBackendsMode(service) {
		provisionedSvcNodes = []*v1.Node{}
	}

	// If network partition, do not proceed
	isNetworkPartition, err := cp.checkForNetworkPartition(logger, nodes, virtualNodeExists)
	if err != nil {
		return err
	} else if isNetworkPartition {
		return nil
	}

	dimensionsMap := make(map[string]string)

	var errorType string
	var lbMetricDimension string

	lb, err := lbProvider.lbClient.GetLoadBalancerByName(ctx, getLoadBalancerCompartment(service, cp.config.CompartmentID), lbName)
	if err != nil && !client.IsNotFound(err) {
		logger.With(zap.Error(err)).Error("Failed to get loadbalancer by name")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		dimensionsMap[metrics.ResourceOCIDDimension] = lbName
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return err
	} else if client.IsNotFound(err) {
		logger.Infof("Could not find load balancer, will not retry UpdateLoadBalancer.")
		return nil
	}

	if lb.LifecycleState == nil || *lb.LifecycleState != lbLifecycleStateActive {
		logger := logger.With("lifecycleState", lb.LifecycleState)
		switch loadBalancerType {
		case NLB:
			// This check is added here since NLBs are marked as failed in case nlb work-requests fail NLB-26239
			if *lb.LifecycleState == string(networkloadbalancer.LifecycleStateFailed) {
				logger.Infof("NLB is in %s state, process the Loadbalancer", *lb.LifecycleState)
			} else {
				return errors.Errorf("NLB is in %s state, wait for NLB to move to %s", *lb.LifecycleState, lbLifecycleStateActive)
			}
			break
		default:
			logger.Infof("LB is not in %s state, will retry UpdateLoadBalancer", lbLifecycleStateActive)
			return errors.Errorf("rejecting request to update LB which is not in %s state", lbLifecycleStateActive)
		}
	}

	lbOCID := ""
	if lb != nil && lb.Id != nil {
		lbOCID = *lb.Id
	} else {
		// if the LB does not exist already use the k8s service UID for reference
		// in logs and metrics
		logger.Error("Load Balancer Id is empty, will retry UpdateLoadBalancer.")
		return errors.New("Load Balancer service returned empty Id, will wait and retry")
	}

	logger = lbProvider.SetLoggerWith("loadBalancerID", lbOCID)
	dimensionsMap[metrics.ResourceOCIDDimension] = lbOCID

	err = lbProvider.checkPendingLBWorkRequests(ctx, lbProvider, lb, service, startTime)
	if err != nil {
		return err
	}

	var sslConfig *SSLConfig
	if requiresCertificate(service) {
		ports, err := getSSLEnabledPorts(service)
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to parse SSL port.")
			errorType = util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
			return err
		}
		secretListenerString := service.Annotations[ServiceAnnotationLoadBalancerTLSSecret]
		secretBackendSetString := service.Annotations[ServiceAnnotationLoadBalancerTLSBackendSetSecret]
		sslConfig = NewSSLConfig(secretListenerString, secretBackendSetString, service, ports, cp)
		// Update SSLConfig from certificate OCID
		if sslConfig, err = updateSSLConfigFromCertOCID(sslConfig, service, logger); err != nil {
			logger.With(zap.Error(err)).Error("Failed to update SSL certificate.")
		}
	}

	lbSubnetIds, err := lbProvider.getLoadBalancerSubnets(ctx, service)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to get Load balancer Subnets.")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return err
	}

	lbSubnets, err := getSubnets(ctx, lbSubnetIds, lbProvider.client.Networking(nil))
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get loadbalancer subnets")
		return err
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, provisionedSvcNodes, cp.client)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get node subnets")
		return err
	}

	ipVersions, err := lbProvider.getOciIpVersions(lbSubnets, nodeSubnets, service)
	if err != nil {
		return err
	}

	spec, err := NewLBSpec(logger, service, provisionedSvcNodes, managedPods, virtualPods, lbSubnetIds, sslConfig, cp.securityListManagerFactory, ipVersions, cp.config.Tags, lb, cp.config.CompartmentID, ipAddressToOcidMap)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to derive LBSpec")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)

		return err
	}
	logger.Info("Updated LB Specs %v", &spec)
	// Existing load balancers cannot change subnets. This ensures that the spec matches
	// what the actual load balancer has listed as the subnet ids. If the load balancer
	// was just created then these values would be equal; however, if the load balancer
	// already existed and the default subnet ids changed, then this would ensure
	// we are setting the security rules on the correct subnets.
	spec, err = updateSpecWithLbSubnets(spec, lb.SubnetIds)
	if err != nil {
		return err
	}

	if err := lbProvider.updateLoadBalancerBackends(ctx, lb, spec); err != nil {
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		logger.With(zap.Error(err)).Error("Failed to update LoadBalancer backends")
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), time.Since(startTime).Seconds(), dimensionsMap)
		return err
	}

	syncTime := time.Since(startTime).Seconds()
	logger.Info("Successfully updated loadbalancer backends")
	lbMetricDimension = util.GetComponentForMetricDimension(util.Success, util.LoadBalancerType)
	dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
	dimensionsMap[metrics.BackendSetsCountDimension] = strconv.Itoa(len(lb.BackendSets))
	metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Update), syncTime, dimensionsMap)
	return nil
}

// getNodesAndPodsByIPs returns slices of Nodes and Pods corresponding to the given IP addresses.
func (clb *CloudLoadBalancerProvider) getNodesAndPodsByIPs(ctx context.Context, cp *CloudProvider, backendIPs []client.IpAddresses, service *v1.Service) ([]*v1.Node, []*v1.Pod, error) {
	ipToPodLookup := make(map[client.IpAddresses]*v1.Pod)
	ipToNodeLookup := make(map[client.IpAddresses]*v1.Node)

	nodeList, err := cp.NodeLister.List(labels.Everything())
	if err != nil {
		return nil, nil, err
	}

	var virtualNodeExists bool
	for _, node := range nodeList {
		if IsVirtualNode(node) {
			virtualNodeExists = true
			continue
		}
		ip := NodeInternalIP(node)
		ipToNodeLookup[ip] = node
	}

	if virtualNodeExists {
		labelSelector := labels.Set(service.Spec.Selector).AsSelector()
		podList, err := cp.PodLister.Pods(service.Namespace).List(labelSelector)
		if err != nil {
			return nil, nil, err
		}
		for i := range podList {
			ips := podList[i].Status.PodIPs
			lookupIP := client.IpAddresses{
				V4: "",
				V6: "",
			}
			for _, ip := range ips {
				if net.IsIPv6String(ip.IP) {
					lookupIP.V6 = ip.IP
				}
				if net.IsIPv4String(ip.IP) {
					lookupIP.V4 = ip.IP
				}
			}
			ipToPodLookup[lookupIP] = podList[i]
		}
	}

	var nodes []*v1.Node
	var pods []*v1.Pod
	for _, ip := range backendIPs {
		if node, nodeExists := ipToNodeLookup[ip]; nodeExists {
			nodes = append(nodes, node)
		} else if pod, podExists := ipToPodLookup[ip]; virtualNodeExists && podExists {
			pods = append(pods, pod)
		} else {
			cp.logger.With("loadBalancerName", GetLoadBalancerName(service), "serviceName", service.Name, "loadBalancerType", getLoadBalancerType(service)).Errorf("provisioned node or virtual pod was not found by IP %q", ip)
		}
	}

	return nodes, pods, nil
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it exists,
// returning nil if the load balancer specified either didn't exist or was
// successfully deleted.
func (cp *CloudProvider) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			log.FromContext(ctx).
				WithValues("component", "cloud-controller-manager").
				Error(err, "Recovered from panic in EnsureLoadBalancerDeleted")
			dimensionsMap := make(map[string]string)
			errorType := util.PANIC
			lbType := strings.ToUpper(getLoadBalancerType(service))
			metricDimension := util.GetComponentForMetricDimension(errorType, lbType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			dimensionsMap[metrics.ClusterOCID] = cp.config.ClusterID
			metrics.SendMetricData(cp.metricPusher, metricDimension, 1, dimensionsMap)
		}
	}()
	startTime := time.Now()
	name := cp.GetLoadBalancerName(ctx, clusterName, service)
	loadBalancerType := getLoadBalancerType(service)

	lbProvider, err := cp.getLoadBalancerProvider(ctx, service)
	if err != nil {
		return errors.Wrap(err, "Unable to get Load Balancer Client.")
	}

	logger := lbProvider.logger
	logger.Debug("Attempting to delete load balancer")
	loadBalancerService := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if acquired := cp.lbLocks.TryAcquire(loadBalancerService); !acquired {
		logger.Error("Could not acquire lock for Deleting Load Balancer")
		return LbOperationAlreadyExists
	}
	defer cp.lbLocks.Release(loadBalancerService)

	var errorType string
	var lbMetricDimension string
	var nsgMetricDimension string

	dimensionsMap := make(map[string]string)
	var frontendNsgId = ""
	uid := fmt.Sprintf("%s", service.UID)
	var etag *string

	securityRuleManagementMode, nsg, err := getRuleManagementMode(service)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get rule management mode")
		return errors.Wrap(err, "failed to get rule management mode")
	}

	lb, err := lbProvider.lbClient.GetLoadBalancerByName(ctx, getLoadBalancerCompartment(service, cp.config.CompartmentID), name)
	if err != nil {
		if client.IsNotFound(err) {
			logger.Info("Could not find load balancer. Nothing to do.")
			if securityRuleManagementMode == NSG {
				displayName := generateNsgName(service)
				nsg.frontendNsgId, etag, err = lbProvider.getFrontendNsgByName(ctx, displayName, getLoadBalancerCompartment(service, cp.config.CompartmentID), cp.config.VCNID, uid)
				if err != nil {
					return errors.Wrap(err, "failed to get frontend NSG")
				}
				// Delete of NSG happens if NSG was created but LB creation fails
				if nsg != nil && nsg.nsgRuleManagementMode == RuleManagementModeNsg && nsg.frontendNsgId != "" {
					if etag != nil {
						logger = lbProvider.SetLoggerWith("frontendNsgId", nsg.frontendNsgId)
						logger.Infof("deleting frontend nsg %s", nsg.frontendNsgId)
						nsgDeleted, err := lbProvider.deleteNsg(ctx, nsg.frontendNsgId, *etag)
						if !nsgDeleted || err != nil {
							logger.With(zap.Error(err)).Error("failed to delete nsg")
							errorType = util.GetError(err)
							nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
							dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
							metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
							return err
						}
						nsgMetricDimension = util.GetComponentForMetricDimension(util.Success, util.NSGType)
						dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
						metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
						logger.Infof("Managed nsg with id %s deleted", nsg.frontendNsgId)
					}
				}
			}
			return nil
		}
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		logger.With(zap.Error(err)).Error("Failed to get loadbalancer by name")
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		dimensionsMap[metrics.ResourceOCIDDimension] = name
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
		return errors.Wrapf(err, "get load balancer %q by name", name)
	}

	id := *lb.Id
	dimensionsMap[metrics.ResourceOCIDDimension] = id
	logger = lbProvider.SetLoggerWith("loadBalancerID", id)

	if securityRuleManagementMode == NSG {
		// List network security groups
		nsgs := lb.NetworkSecurityGroupIds
		for _, nsgId := range nsgs {
			frontendNsgId, etag, err = lbProvider.getFrontendNsg(ctx, nsgId, uid)
			if err != nil {
				errorType = util.GetError(err)
				nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
				dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
				metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Get), time.Since(startTime).Seconds(), dimensionsMap)
			}
			if frontendNsgId != "" {
				logger = lbProvider.SetLoggerWith("frontendNsgId", frontendNsgId)
				nsg.frontendNsgId = frontendNsgId
				break
			}
		}
	}

	// get annotation from load balancer spec and compare to ManagementModeNone
	if securityRuleManagementMode != ManagementModeNone {
		err := lbProvider.cleanupSecurityRulesForLoadBalancerDelete(lb, cp, ctx, service, name, frontendNsgId)
		if err != nil {
			errorType = util.GetError(err)
			lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
			dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Delete), time.Since(startTime).Seconds(), dimensionsMap)

			return err
		}
	}

	logger.Info("Deleting load balancer")
	workReqID, err := lbProvider.lbClient.DeleteLoadBalancer(ctx, id)
	if err != nil {
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		logger.With(zap.Error(err)).Error("Failed to delete loadbalancer")
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Delete), time.Since(startTime).Seconds(), dimensionsMap)

		return errors.Wrapf(err, "delete load balancer %q", id)
	}
	logger.With("workRequestID", workReqID).Info("Await workrequest for delete loadbalancer")
	_, err = lbProvider.lbClient.AwaitWorkRequest(ctx, workReqID)
	if err != nil {
		logger.With(zap.Error(err)).Error("Timeout waiting for loadbalancer delete")
		errorType = util.GetError(err)
		lbMetricDimension = util.GetComponentForMetricDimension(errorType, util.LoadBalancerType)
		dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
		metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
		return errors.Wrapf(err, "awaiting deletion of load balancer %q", name)
	}
	logger.With("workRequestID", workReqID).Info("Workrequest for delete loadbalancer succeeded")
	logger.Info("Loadbalancer deleted")
	lbMetricDimension = util.GetComponentForMetricDimension(util.Success, util.LoadBalancerType)
	dimensionsMap[metrics.ComponentDimension] = lbMetricDimension
	metrics.SendMetricData(cp.metricPusher, getMetric(loadBalancerType, Delete), time.Since(startTime).Seconds(), dimensionsMap)

	// Delete of NSG happens after delete of the Loadbalancer
	if nsg != nil && nsg.nsgRuleManagementMode == RuleManagementModeNsg && nsg.frontendNsgId != "" {
		if etag != nil {
			logger = lbProvider.SetLoggerWith("frontendNsgId", nsg.frontendNsgId)
			logger.Infof("deleting frontend nsg %s", nsg.frontendNsgId)
			nsgDeleted, err := lbProvider.deleteNsg(ctx, nsg.frontendNsgId, *etag)
			if !nsgDeleted || err != nil {
				logger.With(zap.Error(err)).Error("failed to delete nsg")
				errorType = util.GetError(err)
				nsgMetricDimension = util.GetComponentForMetricDimension(errorType, util.NSGType)
				dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
				metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
				return err
			}
			nsgMetricDimension = util.GetComponentForMetricDimension(util.Success, util.NSGType)
			dimensionsMap[metrics.ComponentDimension] = nsgMetricDimension
			metrics.SendMetricData(cp.metricPusher, getMetric(util.NSGType, Delete), time.Since(startTime).Seconds(), dimensionsMap)
			logger.Infof("managed nsg with id %s deleted", nsg.frontendNsgId)
		}
	}

	return nil
}

// Critical Section for Security List Updates
func (clb *CloudLoadBalancerProvider) cleanupSecurityRulesForLoadBalancerDelete(lb *client.GenericLoadBalancer, cp *CloudProvider, ctx context.Context, service *v1.Service, name string, frontendNsgOcid string) error {
	updateRulesMutex.Lock()
	defer updateRulesMutex.Unlock()

	logger := clb.logger
	id := *lb.Id

	ipAddresses := client.IpAddresses{
		V4: "",
		V6: "",
	}
	ipSet := sets.New(ipAddresses)

	for _, backendSet := range lb.BackendSets {
		for _, backend := range backendSet.Backends {
			if net.IsIPv6String(*backend.IpAddress) {
				ipAddresses.V6 = *backend.IpAddress
			} else {
				ipAddresses.V4 = *backend.IpAddress
			}
			ipSet.Insert(ipAddresses)

		}
	}
	nodes, _, err := clb.getNodesAndPodsByIPs(ctx, cp, ipSet.UnsortedList(), service)

	// Reset node list since we need not manage resources for nodes when its Pods as Backend mode
	if isPodsAsBackendsMode(service) {
		nodes = []*v1.Node{}
	}

	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to fetch nodes by internal ips")
		return errors.Wrap(err, "fetching nodes by internal ips")
	}
	nodeSubnets, err := getSubnetsForNodes(ctx, nodes, clb.client)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to get subnets for nodes")
		return errors.Wrap(err, "getting subnets for nodes")
	}

	lbSubnets, err := getSubnets(ctx, lb.SubnetIds, clb.client.Networking(nil))
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to get subnets for load balancers")
		return errors.Wrap(err, "getting subnets for load balancers")
	}

	securityRuleManagerMode, managedNsg, err := getRuleManagementMode(service)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get Security Rule management mode")
		return errors.Wrap(err, "failed to get Security Rule management mode")
	}

	backendNsgIds, err := getManagedBackendNSG(service)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to get backend Nsgs from spec")
		return errors.Wrap(err, "failed to get backend Nsgs from spec")
	}
	var securityListManager securityListManager
	if securityRuleManagerMode == NSG {
		if managedNsg != nil {
			if frontendNsgOcid != "" {
				managedNsg.frontendNsgId = frontendNsgOcid
			}
			if len(backendNsgIds) > 0 {
				managedNsg.backendNsgId = backendNsgIds
			}
		}
	} else {
		securityListManager = cp.securityListManagerFactory(
			securityRuleManagerMode)
	}

	isPreserveSource, err := getPreserveSource(logger, service)
	if err != nil {
		logger.With(zap.Error(err)).Error("failed to determine value for is-preserve-source")
		return errors.Wrap(err, "failed to determine value for is-preserve-source")
	}

	ipVersions, err := clb.getOciIpVersions(lbSubnets, nodeSubnets, service)
	if err != nil {
		return err
	}

	portsNsg, err := getPorts(service, convertOciIpVersionsToOciIpFamilies(ipVersions.ListenerBackendIpVersion), nil, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to get ports from spec")
	}
	sourceCIDRs, err := getLoadBalancerSourceRanges(service)
	if securityRuleManagerMode == NSG && len(managedNsg.backendNsgId) > 0 {
		serviceComponents := securityRuleComponents{
			frontendNsgOcid:  managedNsg.frontendNsgId,
			backendNsgOcids:  managedNsg.backendNsgId,
			sourceCIDRs:      sourceCIDRs,
			ports:            portsNsg,
			isPreserveSource: isPreserveSource,
			serviceUid:       fmt.Sprintf("service-uid-%s", service.UID),
		}
		logger.Infof("(ensureloadbalancer deleted) Service Components %#v", serviceComponents)
		err = clb.removeBackendSecurityGroupRules(ctx, serviceComponents)
		if err != nil {
			return err
		}
	}

	for listenerName, listener := range lb.Listeners {
		backendSetName := *listener.DefaultBackendSetName
		bs, ok := lb.BackendSets[backendSetName]
		if !ok {
			logger.With(zap.Error(err)).Errorf("Failed to delete loadbalencer as backend set %q missing (loadbalancer=%q)", backendSetName, id)
			return errors.Errorf("backend set %q missing (loadbalancer=%q)", backendSetName, id) // Should never happen.
		}

		ports := portsFromBackendSet(clb.logger, backendSetName, &bs)
		ports.ListenerPort = *listener.Port

		logger.With("listenerName", listenerName, "ports", ports).Debug("Deleting security rules for listener")

		sourceCIDRs, err := getLoadBalancerSourceRanges(service)
		if err != nil {
			logger.With(zap.Error(err)).Errorf("Failed to get security rules for listener %q on load balancer %q", listenerName, name)
			return errors.Wrapf(err, "delete security rules for listener %q on load balancer %q", listenerName, name)
		}

		logger.Infof("Security rule management mode %s", securityRuleManagerMode)
		if securityRuleManagerMode == ManagementModeAll || securityRuleManagerMode == ManagementModeFrontend {
			sc := securityRuleComponents{
				lbSubnets:        lbSubnets,
				backendSubnets:   nodeSubnets,
				sourceCIDRs:      sourceCIDRs,
				actualPorts:      nil,
				desiredPorts:     ports,
				isPreserveSource: isPreserveSource,
				ipFamilies:       convertOciIpVersionsToOciIpFamilies(ipVersions.ListenerBackendIpVersion),
			}
			logger.Infof("Service Components security list %#v", sc)
			if err = securityListManager.Delete(ctx, sc); err != nil {
				logger.With(zap.Error(err)).Errorf("Failed to delete security rules for listener %q on load balancer %q", listenerName, name)
				return errors.Wrapf(err, "delete security rules for listener %q on load balancer %q", listenerName, name)
			}
		}

	}
	return nil
}

// only supported by LBaaS
func (clb *CloudLoadBalancerProvider) updateLoadbalancerShape(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	shapeDetails := client.GenericUpdateLoadBalancerShapeDetails{
		ShapeName:    &spec.Shape,
		ShapeDetails: nil,
	}
	if *lb.ShapeName == flexible && spec.Shape != flexible {
		// LBaaS does not support converting from flexible to fixed shapes
		// as that can easily be achieved by setting the min and max bandwith to
		// whatever fixed shape that is needed
		return errors.New("cannot convert LB shape from flexible to fixed shape " + spec.Shape)
	}
	if spec.Shape == flexible {
		shapeDetails.ShapeDetails = &client.GenericShapeDetails{
			MinimumBandwidthInMbps: spec.FlexMin,
			MaximumBandwidthInMbps: spec.FlexMax,
		}
	}
	wrID, err := clb.lbClient.UpdateLoadBalancerShape(ctx, *lb.Id, &shapeDetails)
	if err != nil {
		return errors.Wrap(err, "failed to create UpdateLoadBalancerShape request")
	}
	logger := clb.logger.With("old-shape", *lb.ShapeName, "new-shape", spec.Shape,
		"flexMinimumMbps", spec.FlexMin, "flexMaximumMbps", spec.FlexMax,
		"opc-workrequest-id", wrID, "loadBalancerType", getLoadBalancerType(spec.service))
	logger.Info("Awaiting UpdateLoadBalancerShape workrequest")
	_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return err
	}
	logger.Info("UpdateLoadBalancerShape request completed successfully")
	return nil
}

func (clb *CloudLoadBalancerProvider) updateLoadBalancerNetworkSecurityGroups(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	wrID, err := clb.lbClient.UpdateNetworkSecurityGroups(ctx, *lb.Id, spec.NetworkSecurityGroupIds)
	if err != nil {
		return errors.Wrap(err, "failed to create UpdateNetworkSecurityGroups request")
	}
	logger := clb.logger.With("existingNSGIds", lb.NetworkSecurityGroupIds, "newNSGIds", spec.NetworkSecurityGroupIds,
		"opc-workrequest-id", wrID)
	logger.Info("Awaiting UpdateNetworkSecurityGroups workrequest")
	_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return errors.Wrap(err, "failed to await UpdateNetworkSecurityGroups workrequest")
	}
	logger.Info("Loadbalancer UpdateNetworkSecurityGroups workrequest completed successfully")
	return nil
}

func doesLbHaveOkeSystemTags(lb *client.GenericLoadBalancer, spec *LBSpec) bool {
	if lb.SystemTags == nil || spec.SystemTags == nil {
		return false
	}
	if okeSystemTag, okeSystemTagNsExists := lb.SystemTags[OkeSystemTagNamesapce]; okeSystemTagNsExists {
		return reflect.DeepEqual(okeSystemTag, spec.SystemTags[OkeSystemTagNamesapce])
	}
	return false
}
func (clb *CloudLoadBalancerProvider) addLoadBalancerOkeSystemTags(ctx context.Context, lb *client.GenericLoadBalancer, spec *LBSpec) error {
	lbDefinedTagsRequest := make(map[string]map[string]interface{})

	if _, useWI := spec.service.Annotations[ServiceAnnotationServiceAccountName]; useWI { // When using Workload Identity
		return fmt.Errorf("principal type is workload identity. skip addition of oke system tags.")
	}

	if spec.SystemTags == nil {
		return fmt.Errorf("oke system tag is not found in LB spec. ignoring..")
	}
	if _, exists := spec.SystemTags[OkeSystemTagNamesapce]; !exists {
		return fmt.Errorf("oke system tag namespace is not found in LB spec")
	}

	if lb.DefinedTags != nil {
		lbDefinedTagsRequest = lb.DefinedTags
	}

	// no overwriting customer tags as customer can not have a tag namespace with prefix 'orcl-'
	// system tags are passed as defined tags in the request
	lbDefinedTagsRequest[OkeSystemTagNamesapce] = spec.SystemTags[OkeSystemTagNamesapce]

	// update fails if the number of defined tags is more than the service limit i.e 64
	if len(lbDefinedTagsRequest) > MaxDefinedTagPerResource {
		return fmt.Errorf(MaxDefinedTagErrMessage, spec.Type)
	}

	lbUpdateDetails := &client.GenericUpdateLoadBalancerDetails{
		FreeformTags: lb.FreeformTags,
		DefinedTags:  lbDefinedTagsRequest,
	}
	wrID, err := clb.lbClient.UpdateLoadBalancer(ctx, *lb.Id, lbUpdateDetails)
	if err != nil {
		return errors.Wrap(err, "UpdateLoadBalancer request failed")
	}
	_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return errors.Wrap(err, "failed to await updateloadbalancer work request")
	}

	logger := clb.logger.With("opc-workrequest-id", wrID, "loadBalancerID", lb.Id)
	logger.Info("UpdateLoadBalancer request to add oke system tags completed successfully")
	return nil
}

func (clb *CloudLoadBalancerProvider) updateLoadBalancerIpVersion(ctx context.Context, lb *client.GenericLoadBalancer, details *client.GenericUpdateLoadBalancerDetails) error {
	wrID, err := clb.lbClient.UpdateLoadBalancer(ctx, *lb.Id, details)
	if err != nil {
		return errors.Wrap(err, "failed to create UpdateLoadBalancer request")
	}
	logger := clb.logger.With("existingIpVersion", lb.IpVersion, "newIpVersion", details.IpVersion)
	logger.Infof("Awaiting UpdateLoadBalancer workrequest to update endpoint IpVersion %s", wrID)
	_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return errors.Wrap(err, "failed to await UpdateLoadBalancer workrequest")
	}
	logger.Infof("UpdateLoadBalancer workrequest to update %s endpoint IpVersion completed successfully", *lb.Id)
	return nil
}

func (clb *CloudLoadBalancerProvider) updateLoadbalancerIpVersionTranslation(ctx context.Context, lb *client.GenericLoadBalancer, details *client.GenericUpdateLoadBalancerDetails) error {
	wrID, err := clb.lbClient.UpdateLoadBalancer(ctx, *lb.Id, details)
	if err != nil {
		return errors.Wrap(err, "failed to create UpdateLoadBalancer request")
	}
	clb.logger.Info("Awaiting UpdateLoadBalancer workrequest to update IP version translation mode from %s to %s", lb.IpVersionTranslationMode, details.IpVersionTranslationMode)
	_, err = clb.lbClient.AwaitWorkRequest(ctx, wrID)
	if err != nil {
		return errors.Wrap(err, "failed to await UpdateLoadBalancer workrequest")
	}
	clb.logger.Infof("UpdateLoadBalancer workrequest to update %s IP version translation mode completed successfully", *lb.Id)
	return nil
}

// Given an OCI load balancer, return a LoadBalancerStatus
func loadBalancerToStatus(lb *client.GenericLoadBalancer, ipMode *v1.LoadBalancerIPMode, skipPrivateIp bool, logger *zap.SugaredLogger) (*v1.LoadBalancerStatus, error) {
	// NLB created with an intent to assign an IP address which is already consumed goes into FAILED state
	// An NLB in FAILED state without an IP address is FAILED during create operation
	if len(lb.IpAddresses) == 0 {
		if lb.LifecycleState != nil &&
			(*lb.LifecycleState == string(networkloadbalancer.LifecycleStateFailed) || *lb.LifecycleState == string(loadbalancer.LoadBalancerLifecycleStateFailed)) {
			logger.With("loadBalancerID", *lb.Id).Warnf("Network LoadBalancers without an IP in FAILED state. returning empty LoadBalancerStatus{}")
			return &v1.LoadBalancerStatus{}, nil
		}
		return nil, errors.Errorf("no ip addresses found for load balancer %q", *lb.DisplayName)
	}

	ingress := []v1.LoadBalancerIngress{}
	for _, ip := range lb.IpAddresses {
		if ip.IpAddress == nil {
			continue // should never happen but appears to when EnsureLoadBalancer is called with 0 nodes.
		}

		if skipPrivateIp {
			if !pointer.BoolDeref(ip.IsPublic, false) {
				continue
			}
		}
		ingress = append(ingress, v1.LoadBalancerIngress{IP: *ip.IpAddress, IPMode: ipMode})
	}

	return &v1.LoadBalancerStatus{Ingress: ingress}, nil
}

func (cp *CloudProvider) checkAllBackendNodesNotReady(nodeList []*v1.Node) bool {
	for _, node := range nodeList {
		if _, hasExcludeBalancerLabel := node.Labels[excludeBackendFromLBLabel]; hasExcludeBalancerLabel {
			continue
		}
		for _, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady {
				if cond.Status == v1.ConditionTrue {
					return false
				}
				break
			}
		}
	}
	return true
}

/*
getProvisionedNodesAndPodsOfService return true if Virtual Nodes exist in the cluster and the list of provisioned nodes, managed pods and virtual pods for the service
*/
func (cp *CloudProvider) getProvisionedNodesAndPodsOfService(ctx context.Context, logger *zap.SugaredLogger, service *v1.Service, nodes []*v1.Node) (virtualNodeExists bool, provisionedSvcNodes []*v1.Node, managedPods []*v1.Pod, virtualPods []*v1.Pod, err error) {
	provisionedSvcNodes, err = filterNodes(service, nodes)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to filter provisioned nodes with label selector and virtual nodes")
		return
	}

	// Check if virtual nodes exist in the cluster
	virtualNodeExists, err = VirtualNodeExists(cp.NodeLister)
	if err != nil {
		logger.With(zap.Error(err)).Error("Failed to check if cluster has virtual nodes")
		err = errors.Wrap(err, "failed to check if cluster has virtual nodes")
		return
	}
	isPodsAsBackendsModeEnabled := isPodsAsBackendsMode(service)

	if virtualNodeExists || isPodsAsBackendsModeEnabled {
		// Fetch managed and virtual pods based on the service Endpointslices
		managedPods, virtualPods, err = cp.getManagedAndVirtualPodsOfService(ctx, logger, service)
		if !virtualNodeExists {
			// Should ideally be already nil if Virtual Nodes don't exist, just in case it isn't while moving between vanilla OKE, mixed OKE and SKE
			virtualPods = nil
		}
		if !isPodsAsBackendsMode(service) {
			// Managed pods are never backends unless it is the Pods as Backends Mode of LB management
			managedPods = nil
		}
	}

	return
}

func isPodsAsBackendsMode(service *v1.Service) bool {
	return reflect.DeepEqual(service.Spec.AllocateLoadBalancerNodePorts, pointer.Bool(false)) && npnEnabled && !hasCustomNodePorts(service)
}

// getManagedAndVirtualPodsOfService returns pods scheduled on managed nodes and pods scheduled on virtual nodes fronted by the given Service
func (cp *CloudProvider) getManagedAndVirtualPodsOfService(ctx context.Context, logger *zap.SugaredLogger, service *v1.Service) ([]*v1.Pod, []*v1.Pod, error) {
	endpointSlices, err := cp.getEndpointSlicesForService(service)
	if err != nil {
		return nil, nil, err
	}

	endpointSet := make(map[string]struct{})
	virtualPodsSet := make(map[string]*v1.Pod)
	managedPodsSet := make(map[string]*v1.Pod)
	var virtualPods []*v1.Pod
	var managedPods []*v1.Pod
	for _, es := range endpointSlices {
		for _, e := range es.Endpoints {
			if e.TargetRef == nil || e.TargetRef.Kind != "Pod" {
				// Endpoint target is not a pod, ignore
				continue
			}
			if e.Addresses == nil || len(e.Addresses) < 1 {
				// Endpoint does not have an address, ignore
				continue
			}
			if _, exists := endpointSet[e.Addresses[0]]; exists {
				// Endpoint already considered, skip
				continue
			}

			pod, err := cp.PodLister.Pods(es.Namespace).Get(e.TargetRef.Name)
			if err != nil {
				if apierrors.IsNotFound(err) {
					logger.With(zap.Error(err)).Errorf("Pod object does not exist: %s", e.TargetRef.Name)
					continue
				}
				return nil, nil, err
			}

			node, err := cp.NodeLister.Get(pod.Spec.NodeName)
			if err != nil {
				if apierrors.IsNotFound(err) {
					logger.With(zap.Error(err)).Errorf("Node object does not exist: %s", pod.Spec.NodeName)
					continue
				}
				return nil, nil, err
			}

			if IsVirtualNode(node) {
				virtualPodsSet[string(pod.UID)] = pod
			} else {
				// Managed Node
				managedPodsSet[string(pod.UID)] = pod
			}

			endpointSet[e.Addresses[0]] = struct{}{}
		}
	}
	// Return a list of unique pods
	for _, pod := range virtualPodsSet {
		virtualPods = append(virtualPods, pod)
	}
	for _, pod := range managedPodsSet {
		managedPods = append(managedPods, pod)
	}
	return managedPods, virtualPods, nil
}

func (cp *CloudProvider) getEndpointSlicesForService(service *v1.Service) ([]*discovery.EndpointSlice, error) {
	esLabelSelector := labels.Set(map[string]string{
		discovery.LabelServiceName: service.Name,
	}).AsSelectorPreValidated()

	endpointSlices, err := cp.EndpointSliceLister.EndpointSlices(service.Namespace).List(esLabelSelector)
	if err != nil {
		return nil, err
	}

	return endpointSlices, nil
}

// If CCM manages the NSG for the service, CCM to delete the NSG when the LB/NLB service is deleted
func (clb *CloudLoadBalancerProvider) deleteNsg(ctx context.Context, id, etag string) (bool, error) {
	opcRequestId, err := clb.client.Networking(nil).DeleteNetworkSecurityGroup(ctx, id, etag)
	if err != nil {
		clb.logger.Errorf("failed to delete nsg %s", id)
		return false, err
	}
	clb.logger.Infof("delete nsg OpcRequestId %s", pointer.StringDeref(opcRequestId, ""))
	return true, nil
}

func (clb *CloudLoadBalancerProvider) getFrontendNsg(ctx context.Context, id, uid string) (frontendNsgId string, etag *string, err error) {
	nsg, etag, err := clb.client.Networking(nil).GetNetworkSecurityGroup(ctx, id)
	if err != nil || nsg == nil || etag == nil {
		clb.logger.Errorf("failed to get nsg %s", id)
		return "", nil, err
	}
	freeFormTags := map[string]string{client.CreatedBy: client.CCM, "ServiceUid": uid}
	if reflect.DeepEqual(nsg.FreeformTags, freeFormTags) {
		nsgId := pointer.StringDeref(nsg.Id, "")
		clb.logger.Infof("Found managed frontend nsg %s", nsgId)
		return nsgId, etag, nil
	} else {
		clb.logger.Infof("Found attached nsgs %s but not managed", pointer.StringDeref(nsg.Id, ""))
		return "", nil, nil
	}
}

func (clb *CloudLoadBalancerProvider) getFrontendNsgByName(ctx context.Context, displayName, compartmentId, vcnId, uid string) (frontendNsgId string, etag *string, err error) {
	nsgs, err := clb.client.Networking(nil).ListNetworkSecurityGroups(ctx, displayName, compartmentId, vcnId)
	for _, nsg := range nsgs {
		frontendNsgId, etag, err = clb.getFrontendNsg(ctx, pointer.StringDeref(nsg.Id, ""), uid)
		if err != nil {
			return "", nil, err
		}
		if frontendNsgId != "" {
			clb.logger.Infof("found frontend NSG %s", frontendNsgId)
			return frontendNsgId, etag, nil
		}
	}
	return "", nil, nil
}

// checkPendingLBWorkRequests checks if we have pending work requests before processing the LoadBalancer further
// Will error out if any in-progress work request are present for the LB
func (clb *CloudLoadBalancerProvider) checkPendingLBWorkRequests(ctx context.Context, lbProvider CloudLoadBalancerProvider, lb *client.GenericLoadBalancer, service *v1.Service, startTime time.Time) (err error) {
	listWorkRequestTime := time.Now()
	loadBalancerType := getLoadBalancerType(service)

	switch loadBalancerType {
	case NLB:
		if *lb.LifecycleState == string(networkloadbalancer.LifecycleStateUpdating) {
			clb.logger.Info("Load Balancer is in UPDATING state, possibly a work request is in progress")
			return errors.New("Load Balancer might have work requests in progress, will wait and retry")
		}
	default:
		lbInProgressWorkRequests, err := lbProvider.lbClient.ListWorkRequests(ctx, *lb.CompartmentId, *lb.Id)
		clb.logger.Infof("time (in seconds) to list work-requests for LB %f", time.Since(listWorkRequestTime).Seconds())
		if err != nil {
			clb.logger.With(zap.Error(err)).Error("Failed to list work-requests in-progress")
			return err
		}
		for _, wr := range lbInProgressWorkRequests {
			if *wr.LifecycleState == string(loadbalancer.WorkRequestLifecycleStateInProgress) || *wr.LifecycleState == string(loadbalancer.WorkRequestLifecycleStateAccepted) {
				clb.logger.Infof("current in-progress work requests for Load Balancer %s", *wr.Id)
				return errors.New("Load Balancer has work requests in progress, will wait and retry")
			}
		}
	}
	return
}

// checkForNetworkPartition return true if network partition is present (all nodes are not ready) else throws an error if any
func (cp *CloudProvider) checkForNetworkPartition(logger *zap.SugaredLogger, nodes []*v1.Node, virtualNodeExists bool) (isNetworkPartition bool, err error) {
	// Service controller provided empty provisioned nodes list
	if len(nodes) == 0 && !virtualNodeExists {
		// List all nodes in the cluster
		nodeList, err := cp.NodeLister.List(labels.Everything())
		if err != nil {
			logger.With(zap.Error(err)).Error("Failed to check if all backend nodes are not ready, error listing nodes")
			return false, err
		}

		// Filter out only provisioned nodes
		nodeList = filterProvisionedNodes(nodeList)

		if len(nodeList) == 0 {
			logger.Info("Cluster has zero nodes, continue reconciling")
		} else if allNodesNotReady := cp.checkAllBackendNodesNotReady(nodeList); allNodesNotReady {
			logger.Info("Not removing backends since all nodes are Not Ready")
			return true, nil
		} else {
			err = errors.Errorf("backend node status is inconsistent, will retry")
			logger.With(zap.Error(err)).Error("Not removing backends since backend node status is inconsistent with what was observed by service controller")
			return false, err
		}
	}
	return
}

func (clb *CloudLoadBalancerProvider) getLbEndpointIpVersion(ipFamilies []string, ipFamilyPolicy string, lbSubnets []*core.Subnet, isTranslationEnabled bool) (string, error) {
	lbEndpointVersion := ""
	errIPv6Subnet := checkSubnetIpFamilyCompatibility(lbSubnets, IPv6)
	errIPv4Subnet := checkSubnetIpFamilyCompatibility(lbSubnets, IPv4)
	SingleStackIPv4 := "SingleStackIPv4"
	SingleStackIPv6 := "SingleStackIPv6"
	DualStack := "DualStack"
	switch ipFamilyPolicy {
	case string(v1.IPFamilyPolicySingleStack):
		if ipFamilies[0] == IPv6 {
			if errIPv6Subnet != nil {
				return "", errors.Wrapf(errIPv6Subnet, "subnet does not have %s CIDR blocks", IPv6)
			}
			lbEndpointVersion = SingleStackIPv6
		}
		if ipFamilies[0] == IPv4 {
			if errIPv4Subnet != nil {
				return "", errors.Wrapf(errIPv4Subnet, "subnet does not have %s CIDR blocks", IPv4)
			}
			lbEndpointVersion = SingleStackIPv4
		}
	case string(v1.IPFamilyPolicyRequireDualStack):
		if errIPv6Subnet != nil {
			return "", errors.Wrapf(errIPv6Subnet, "subnet does not have %s CIDR blocks", IPv6)
		}
		if errIPv4Subnet != nil {
			return "", errors.Wrapf(errIPv4Subnet, "subnet does not have %s CIDR blocks", IPv4)
		}
		lbEndpointVersion = DualStack
	case string(v1.IPFamilyPolicyPreferDualStack):
		lbEndpointVersion = DualStack
		if errIPv4Subnet != nil && errIPv6Subnet != nil {
			// This should never happen
			return "", errors.New("subnet does not have IPv4 or IPv6 cidr, can't create loadbalancer")
		}
		// make a hard check on subnet compatibility for dual stack even incase it is preferred.
		if isTranslationEnabled {
			if errIPv6Subnet != nil || errIPv4Subnet != nil {
				return "", errors.New("IP version translation is enabled but subnet is compatible for dual stack")
			}
		}
		if errIPv6Subnet != nil {
			clb.logger.Warn("subnet provided does not have IPv6 subnet CIDR block, creating LB with only IPv4 endpoint")
			lbEndpointVersion = SingleStackIPv4
		}
		if errIPv4Subnet != nil {
			clb.logger.Warn("subnet provided does not have IPV4 subnet CIDR block, creating LB with only IPv6 endpoint")
			lbEndpointVersion = SingleStackIPv6
		}
	default:
		lbEndpointVersion = SingleStackIPv4
	}
	if strings.Compare(lbEndpointVersion, SingleStackIPv4) == 0 {
		lbEndpointVersion = IPv4
	}
	if strings.Compare(lbEndpointVersion, SingleStackIPv6) == 0 {
		lbEndpointVersion = IPv6
	}
	if strings.Compare(lbEndpointVersion, DualStack) == 0 {
		lbEndpointVersion = IPv4AndIPv6
	}
	return lbEndpointVersion, nil
}

func (clb *CloudLoadBalancerProvider) getLbListenerBackendSetIpVersion(ipFamilies []string, ipFamilyPolicy string, nodeSubnets []*core.Subnet) ([]string, error) {
	errIPv6Subnet := checkSubnetIpFamilyCompatibility(nodeSubnets, IPv6)
	errIPv4Subnet := checkSubnetIpFamilyCompatibility(nodeSubnets, IPv4)
	switch ipFamilyPolicy {
	case string(v1.IPFamilyPolicySingleStack):
		if ipFamilies[0] == IPv6 {
			if errIPv6Subnet != nil {
				return []string{}, errors.Wrapf(errIPv6Subnet, "subnet does not have %s CIDR blocks", IPv6)
			}
			return []string{IPv6}, nil
		}
		if ipFamilies[0] == IPv4 {
			if errIPv4Subnet != nil {
				return []string{}, errors.Wrapf(errIPv4Subnet, "subnet does not have %s CIDR blocks", IPv4)
			}
			return []string{IPv4}, nil
		}
	case string(v1.IPFamilyPolicyRequireDualStack):
		if errIPv6Subnet != nil {
			return []string{}, errors.Wrapf(errIPv6Subnet, "subnet does not have %s CIDR blocks", IPv6)
		}
		if errIPv4Subnet != nil {
			return []string{}, errors.Wrapf(errIPv4Subnet, "subnet does not have %s CIDR blocks", IPv4)
		}
		return []string{IPv4, IPv6}, nil
	case string(v1.IPFamilyPolicyPreferDualStack):
		if errIPv6Subnet != nil && errIPv4Subnet != nil {
			// should never happen
			return nil, errors.New("subnet does not have both IPv4 and IPv6 cidr, can't create loadbalancer")
		}
		if errIPv6Subnet != nil {
			clb.logger.Warn("subnet provided does not have IPv6 subnet CIDR block, creating listeners and backends of ip-version IPv4")
			return []string{IPv4}, nil
		}
		if errIPv4Subnet != nil {
			clb.logger.Warn("subnet provided does not have IPV4 subnet CIDR block, creating listeners and backends of ip-version IPv6")
			return []string{IPv6}, nil
		}
		return []string{IPv4, IPv6}, nil
	}
	return []string{IPv4}, nil
}

func (clb *CloudLoadBalancerProvider) getOciIpVersions(lbSubnets, nodeSubnets []*core.Subnet, service *v1.Service) (*IpVersions, error) {
	ipFamilies := getIpFamilies(service)
	ipFamilyPolicy := getIpFamilyPolicy(service)

	lbEndpointVersion, err := clb.getLbEndpointIpVersion(ipFamilies, ipFamilyPolicy, lbSubnets, isNat46Enabled(service))
	if err != nil {
		return nil, err
	}
	listenerBackendIpVersion, err := clb.getLbListenerBackendSetIpVersion(ipFamilies, ipFamilyPolicy, nodeSubnets)
	if err != nil {
		return nil, err
	}

	if getLoadBalancerType(service) == LB {
		if lbEndpointVersion == IPv6 {
			return nil, errors.New("SingleStack IPv6 is not supported for OCI LBaaS")
		}
		listenerBackendIpVersion = []string{IPv4}
	}

	var ociListenerBackendIpVersion []client.GenericIpVersion
	for _, ipFamily := range listenerBackendIpVersion {
		ociListenerBackendIpVersion = append(ociListenerBackendIpVersion, convertK8sIpFamiliesToOciIpVersion(ipFamily))
	}
	ipVersions := &IpVersions{
		IpFamilies:               ipFamilies,
		IpFamilyPolicy:           common.String(ipFamilyPolicy),
		LbEndpointIpVersion:      GenericIpVersion(convertK8sIpFamiliesToOciIpVersion(lbEndpointVersion)),
		ListenerBackendIpVersion: ociListenerBackendIpVersion,
	}
	return ipVersions, nil
}

func (clb *CloudLoadBalancerProvider) SetLogger(logger *zap.SugaredLogger) {
	clb.logger = logger
}

func (clb *CloudLoadBalancerProvider) SetLoggerWith(args ...interface{}) *zap.SugaredLogger {
	clb.logger = clb.logger.With(args...)
	return clb.logger
}

func hasCustomNodePorts(service *v1.Service) bool {
	for _, port := range service.Spec.Ports {
		if port.NodePort != 0 {
			return true
		}
	}
	return false
}

// getIpAddressOcidMap returns a map of IP address <-> ocid of IP address, for all the IPs on the primary vnics of the nodes
// used to construct backends for the managed Nodes.
// TODO: the method should handle for virtualNodes, ManagedPods ..
func (cp *CloudProvider) getIpAddressOcidMap(ctx context.Context, provisionedSvcNodes []*v1.Node) (map[string]string, error) {
	ipAddressOcidMap := make(map[string]string)
	var addresses []v1.NodeAddress
	for _, node := range provisionedSvcNodes {
		id, err := MapProviderIDToResourceID(node.Spec.ProviderID)
		if err != nil {
			return ipAddressOcidMap, err
		}

		addresses = node.Status.Addresses
		if addresses == nil {
			addresses, err = cp.NodeAddressesByProviderID(ctx, id)
		}

		for _, address := range addresses {
			if net.IsIPv4String(address.Address) {
				ipAddressOcidMap[address.Address] = id
			}
			if net.IsIPv6String(address.Address) {
				ipv6Id, err := cp.getIPv6IdByAddress(ctx, address.Address, id)
				if err != nil {
					return nil, err
				}
				ipAddressOcidMap[address.Address] = ipv6Id
			}
		}
	}
	return ipAddressOcidMap, nil
}

func updateSSLConfigFromCertOCID(sslConfig *SSLConfig, service *v1.Service, logger *zap.SugaredLogger) (*SSLConfig, error) {
	// SSL Config update with listeners port sslConfigurationDetails as certificate OCID
	if _, ok := service.Annotations[ServiceAnnotationLoadBalancerCertificateOcid]; ok {
		ports, err := getSSLEnabledPorts(service)
		if err != nil {
			return nil, err
		}
		ocid, _ := getTlsCertificateOCID(service)
		listenerTlsConfigMap := make(map[int]string)
		for _, port := range ports {
			// CertificateIds are supported in oci sdk version v65
			listenerTlsConfigMap[port] = ocid
		}
		logger.Info("Listener tlsconfig map", listenerTlsConfigMap)
		// Update Listener TLS in SSL Config
		sslBuilder := &SSLConfigBuilder{sslConfig: sslConfig}
		sslConfig = sslBuilder.WithListenerTls(listenerTlsConfigMap).Build()
	}
	return sslConfig, nil
}
