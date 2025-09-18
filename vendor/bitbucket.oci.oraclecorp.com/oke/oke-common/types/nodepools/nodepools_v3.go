package nodepools

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/oracle/oci-go-sdk/v65/common"
	"sort"
	"strings"

	"bitbucket.oci.oraclecorp.com/oke/oke-common/protobuf"
	nodes "bitbucket.oci.oraclecorp.com/oke/oke-common/types/nodes"
)

// Set of constants representing the allowable values for NodeSource
const (
	NodeSourceTypeImage string = "IMAGE"
)

// GetResponseV3 is the response type for the NodePoolList CLI API operation.
// TODO: This is not using the correct schema for the NodePoolList operation. It is calling /api/20180228/nodePools but should be using /api/20180222/nodePools
type GetResponseV3 struct {
	NodePools []NodePoolV3
}

// ToV3 converts a nodepools.GetResponse object to a NodePoolGetResponseV3 object understood by the higher layers
func (src *GetResponse) ToGetResponseV3(exposeFaultDomainAndPrivateIp bool, enableNodePoolEnhancements bool) GetResponseV3 {
	return toV3(src, exposeFaultDomainAndPrivateIp, enableNodePoolEnhancements)
}

func toV3(src *GetResponse, exposeFaultDomainAndPrivateIp bool, enableNodePoolEnhancements bool) GetResponseV3 {
	dst := GetResponseV3{
		NodePools: make([]NodePoolV3, 0),
	}
	if src == nil {
		return dst
	}
	for _, np := range src.NodePools {
		dst.NodePools = append(dst.NodePools, np.ToNodePoolV3(exposeFaultDomainAndPrivateIp, enableNodePoolEnhancements))
	}
	return dst
}

// NodePoolSummaryV3 is an array element in the response type for the ListNodePools API operation.
type NodePoolSummaryV3 struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	CompartmentID     string                     `json:"compartmentId"`
	ClusterID         string                     `json:"clusterId"`
	KubernetesVersion string                     `json:"kubernetesVersion"`
	NodeImageID       string                     `json:"nodeImageId"`
	NodeImageName     string                     `json:"nodeImageName"`
	NodeSource        NodeSourceOption           `json:"nodeSource,omitempty"`
	NodeSourceDetails NodeSourceDetails          `json:"nodeSourceDetails,omitempty"`
	NodeShape         string                     `json:"nodeShape"`
	NodeShapeConfig   *NodeShapeConfig           `json:"nodeShapeConfig,omitempty"`
	InitialNodeLabels *[]KeyValueV3              `json:"initialNodeLabels,omitempty"`
	SSHPublicKey      string                     `json:"sshPublicKey"`
	QuantityPerSubnet uint32                     `json:"quantityPerSubnet"`
	SubnetIDs         []string                   `json:"subnetIds"`
	NodeMetadata      map[string]string          `json:"nodeMetadata,omitempty"`
	NodeConfigDetails *NodePoolNodeConfigDetails `json:"nodeConfigDetails,omitempty"`
}

type NodeSourceOption interface {
	GetSourceName() *string
}

type NodeSourceViaImageOption struct {
	SourceName *string `mandatory:"false" json:"sourceName"`
	ImageId    *string `mandatory:"false" json:"imageId"`
}

//GetSourceName returns SourceName
func (m NodeSourceViaImageOption) GetSourceName() *string {
	return m.SourceName
}

func (m NodeSourceViaImageOption) ToInterface() interface{} {
	return m
}

func (m NodeSourceViaImageOption) MarshalJSON() (buff []byte, e error) {
	type MarshalTypeNodeSourceViaImageOption NodeSourceViaImageOption
	s := struct {
		DiscriminatorParam string `json:"sourceType"`
		MarshalTypeNodeSourceViaImageOption
	}{
		"IMAGE",
		(MarshalTypeNodeSourceViaImageOption)(m),
	}

	return json.Marshal(&s)
}

// ToNodePoolSummaryV3s converts a nodepools.GetResponse to a NodePoolSummaryV3 object understood by the higher layers
func (src *GetResponse) ToNodePoolSummaryV3s(enableNodePoolEnhancements bool) []NodePoolSummaryV3 {
	return toSummaryV3(src, enableNodePoolEnhancements)
}

func toSummaryV3(src *GetResponse, enableNodePoolEnhancements bool) []NodePoolSummaryV3 {
	resp := []NodePoolSummaryV3{}
	if src == nil {
		return resp
	}

	for _, np := range src.NodePools {
		resp = append(resp, np.toSummaryV3(enableNodePoolEnhancements))
	}
	return resp
}

// ToSummaryV3 converts a nodepools.GetResponse to a NodePoolSummaryV3 object understood by the higher layers
func (src *NodePool) toSummaryV3(enableNodePoolEnhancements bool) NodePoolSummaryV3 {
	dst := NodePoolSummaryV3{}

	if src == nil {
		return dst
	}

	dst.ID = src.ID
	dst.Name = src.Name
	dst.CompartmentID = src.CompartmentID
	dst.ClusterID = src.ClusterID
	dst.KubernetesVersion = src.K8SVersion
	dst.NodeImageID = src.NodeImageID
	dst.NodeImageName = src.NodeImageName
	dst.NodeShape = src.NodeShape
	initialNodeLabels := KeyValuesFromString(src.InitialNodeLabels)
	dst.InitialNodeLabels = &initialNodeLabels
	dst.SSHPublicKey = src.SSHPublicKey
	if src.NodeOcpus != nil || src.NodeMemoryInGBs != nil {
		dst.NodeShapeConfig = &NodeShapeConfig{}
	}
	if src.NodeOcpus != nil {
		ocpus := src.NodeOcpus.GetValue()
		dst.NodeShapeConfig.Ocpus = &ocpus
	}
	if src.NodeMemoryInGBs != nil {
		memoryInGBs := src.NodeMemoryInGBs.GetValue()
		dst.NodeShapeConfig.MemoryInGBs = &memoryInGBs
	}

	if src.NodeImageID != "" {
		nodeSourceImageOption := NodeSourceViaImageOption{
			SourceName: common.String(src.NodeImageName),
			ImageId:    common.String(src.NodeImageID),
		}.ToInterface()
		dst.NodeSource = nodeSourceImageOption.(NodeSourceOption)

		if enableNodePoolEnhancements {
			nodeSourceDetails := NodeSourceViaImageDetails{
				SourceType: NodeSourceTypeImage,
				ImageId: common.String(src.NodeImageID),
			}
			if src.NodeBootVolumeSizeInGBs != nil {
				nodeSourceDetails.BootVolumeSizeInGBs = &src.NodeBootVolumeSizeInGBs.Value
			}
			dst.NodeSourceDetails = nodeSourceDetails.ToInterface().(NodeSourceDetails)
		}
	} else {
		dst.NodeSource = nil
		dst.NodeSourceDetails = nil
	}

	// fill in Size/placementConfigs/SubnetIDs all the time
	// fill in QuantityPerSubnet if QuantityPerSubnet > 0
	dst.NodeConfigDetails = &NodePoolNodeConfigDetails{
		PlacementConfigs: convertSubnetInfoToPlacementConfigs(src.SubnetsInfo)}
	dst.SubnetIDs = convertSubnetInfoToUniqueSubnetIds(src.SubnetsInfo)
	if src.Size > 0 {
		dst.NodeConfigDetails.Size = src.Size
		dst.QuantityPerSubnet = 0
	} else { // QuantityPerSubnet > 0
		dst.NodeConfigDetails.Size = src.QuantityPerSubnet * uint32(len(src.SubnetsInfo))
		dst.QuantityPerSubnet = src.QuantityPerSubnet
	}
	dst.NodeConfigDetails.NsgIds = src.NsgIds

	dst.NodeMetadata = make(map[string]string)
	for k, v := range src.NodeMetadata {
		dst.NodeMetadata[k] = v
	}

	return dst
}

// NodePoolV3 is the response type for the GetNodePool API operation.
type NodePoolV3 struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	CompartmentID     string                     `json:"compartmentId"`
	ClusterID         string                     `json:"clusterId"`
	KubernetesVersion string                     `json:"kubernetesVersion"`
	NodeImageID       string                     `json:"nodeImageId"`
	NodeImageName     string                     `json:"nodeImageName"`
	NodeSource        NodeSourceOption           `json:"nodeSource,omitempty"`
	NodeSourceDetails NodeSourceDetails          `json:"nodeSourceDetails,omitempty"`
	NodeShape         string                     `json:"nodeShape"`
	NodeShapeConfig   *NodeShapeConfig           `json:"nodeShapeConfig,omitempty"`
	InitialNodeLabels *[]KeyValueV3              `json:"initialNodeLabels,omitempty"`
	SSHPublicKey      string                     `json:"sshPublicKey"`
	QuantityPerSubnet uint32                     `json:"quantityPerSubnet"`
	SubnetIDs         []string                   `json:"subnetIds"`
	Nodes             []nodes.NodeV3             `json:"nodes,omitempty"`
	NodeMetadata      map[string]string          `json:"nodeMetadata,omitempty"`
	NodeConfigDetails *NodePoolNodeConfigDetails `json:"nodeConfigDetails,omitempty"`
}

// ToV3 converts a NodePool object to a NodePoolV3 object understood by the higher layers
// exposeFaultDomainAndPrivateIp is a whitelist flag
func (np *NodePool) ToNodePoolV3(exposeFaultDomainAndPrivateIp bool, enableNodePoolEnhancements bool) NodePoolV3 {
	return toNodePoolV3(np, exposeFaultDomainAndPrivateIp, enableNodePoolEnhancements)
}

func toNodePoolV3(src *NodePool, exposeFaultDomainAndPrivateIp bool, enableNodePoolEnhancements bool) NodePoolV3 {
	dst := NodePoolV3{}
	if src == nil {
		return dst
	}
	dst.ID = src.ID
	dst.Name = src.Name
	dst.CompartmentID = src.CompartmentID
	dst.ClusterID = src.ClusterID
	dst.KubernetesVersion = src.K8SVersion
	dst.NodeImageID = src.NodeImageID
	dst.NodeImageName = src.NodeImageName
	dst.NodeShape = src.NodeShape
	initialNodeLabels := KeyValuesFromString(src.InitialNodeLabels)
	dst.InitialNodeLabels = &initialNodeLabels
	dst.SSHPublicKey = src.SSHPublicKey
	if src.NodeOcpus != nil || src.NodeMemoryInGBs != nil {
		dst.NodeShapeConfig = &NodeShapeConfig{}
	}
	if src.NodeOcpus != nil {
		ocpus := src.NodeOcpus.GetValue()
		dst.NodeShapeConfig.Ocpus = &ocpus
	}
	if src.NodeMemoryInGBs != nil {
		memoryInGBs := src.NodeMemoryInGBs.GetValue()
		dst.NodeShapeConfig.MemoryInGBs = &memoryInGBs
	}


	if src.NodeImageID != "" {
		nodeSourceImageOption := NodeSourceViaImageOption{
			SourceName: common.String(src.NodeImageName),
			ImageId:    common.String(src.NodeImageID),
		}.ToInterface()

		dst.NodeSource = nodeSourceImageOption.(NodeSourceOption)
		if enableNodePoolEnhancements {
			nodeSourceDetails := NodeSourceViaImageDetails{
				SourceType: NodeSourceTypeImage,
				ImageId: common.String(src.NodeImageID),
			}
			if src.NodeBootVolumeSizeInGBs != nil {
				nodeSourceDetails.BootVolumeSizeInGBs = &src.NodeBootVolumeSizeInGBs.Value
			}
			dst.NodeSourceDetails = nodeSourceDetails.ToInterface().(NodeSourceDetails)
		}
	} else {
		dst.NodeSource = nil
		dst.NodeSourceDetails = nil
	}

	// fill in SubnetIDs all the time
	// fill in QuantityPerSubnet if QuantityPerSubnet exists
	dst.NodeConfigDetails = &NodePoolNodeConfigDetails{
		PlacementConfigs: convertSubnetInfoToPlacementConfigs(src.SubnetsInfo)}
	dst.SubnetIDs = convertSubnetInfoToUniqueSubnetIds(src.SubnetsInfo)
	if src.Size > 0 {
		dst.NodeConfigDetails.Size = src.Size
		dst.QuantityPerSubnet = 0
	} else { // QuantityPerSubnet >= 0
		dst.NodeConfigDetails.Size = src.QuantityPerSubnet * uint32(len(src.SubnetsInfo))
		dst.QuantityPerSubnet = src.QuantityPerSubnet
	}
	dst.NodeConfigDetails.NsgIds = src.NsgIds

	for _, nd := range src.NodeStates {
		dst.Nodes = append(dst.Nodes, nd.ToV3(exposeFaultDomainAndPrivateIp))
	}

	dst.NodeMetadata = make(map[string]string)
	for k, v := range src.NodeMetadata {
		dst.NodeMetadata[k] = v
	}

	return dst
}

// CreateNodePoolDetailsV3 is the request body for a CreateNodePool operation.
type CreateNodePoolDetailsV3 struct {
	Name              string                    `json:"name"`
	CompartmentID     string                    `json:"compartmentId"`
	ClusterID         string                    `json:"clusterId"`
	KubernetesVersion string                    `json:"kubernetesVersion"`
	NodeImageName     string                    `json:"nodeImageName"`
	NodeSourceDetails CreateNodeSourceDetails   `json:"nodeSourceDetails,omitempty"`
	NodeShape         string                    `json:"nodeShape"`
	NodeShapeConfig   *NodeShapeConfig           `json:"nodeShapeConfig,omitempty"`
	NodeMetadata      map[string]string         `json:"nodeMetadata"`
	InitialNodeLabels []KeyValueV3              `json:"initialNodeLabels,omitempty"`
	SSHPublicKey      string                    `json:"sshPublicKey"`
	QuantityPerSubnet uint32                    `json:"quantityPerSubnet"`
	SubnetIDs         []string                  `json:"subnetIds"`
	NodeConfigDetails NodePoolNodeConfigDetails `json:"nodeConfigDetails,omitempty"`
}

// Use CreateNodeSourceDetails to enable polymorphic unmarshaling on Create/Update
type CreateNodeSourceDetails struct {
	JsonData   []byte
	SourceType string `json:"sourceType"`
}

// NodeShapeConfig has the same structure for Create/Update/Read/List, so use the same one for now.
type NodeShapeConfig struct {
	Ocpus *float32 `json:"ocpus,omitempty"`
	MemoryInGBs *float32 `json:"memoryInGBs,omitempty"`
}

type CreateNodeSourceViaImageDetails struct {
	ImageId *string `mandatory:"true" json:"imageId"`
	BootVolumeSizeInGBs *uint32 `json:"bootVolumeSizeInGBs,omitempty"`
}

func (m *CreateNodeSourceDetails) UnmarshalJSON(data []byte) error {
	m.JsonData = data
	type Unmarshalernodesourcedetails CreateNodeSourceDetails
	s := struct {
		Model Unmarshalernodesourcedetails
	}{}
	err := json.Unmarshal(data, &s.Model)
	if err != nil {
		return err
	}
	m.SourceType = s.Model.SourceType

	return err
}

func (m *CreateNodeSourceDetails) UnmarshalPolymorphicJSON(data []byte) (interface{}, error) {
	if data == nil || string(data) == "null" {
		return nil, nil
	}

	var err error
	switch m.SourceType {
	case NodeSourceTypeImage:
		mm := CreateNodeSourceViaImageDetails{}
		err = json.Unmarshal(data, &mm)
		return mm, err
	default:
		return *m, nil
	}
}

// Use NodeSourceDetails on NodePool GET requests
type NodeSourceDetails interface {
}

type NodeSourceViaImageDetails struct {
	SourceType string `mandatory:"true" json:"sourceType"`
	ImageId *string `mandatory:"true" json:"imageId"`
	BootVolumeSizeInGBs *uint32 `json:"bootVolumeSizeInGBs,omitempty"`
}

func (m NodeSourceViaImageDetails) ToInterface() interface{} {
	return m
}

// UpdateNodePoolDetailsV3 is the request body for a UpdateNodePool operation.
type UpdateNodePoolDetailsV3 struct {
	Name              string                    `json:"name"`
	KubernetesVersion string                    `json:"kubernetesVersion"`
	QuantityPerSubnet uint32                    `json:"quantityPerSubnet"`
	InitialNodeLabels []KeyValueV3              `json:"initialNodeLabels,omitempty"`
	SubnetIDs         []string                  `json:"subnetIds"`
	NodeConfigDetails NodePoolNodeConfigDetails `json:"nodeConfigDetails,omitempty"`
	NodeMetadata      map[string]string         `json:"nodeMetadata,omitempty"`
	NodeShape         string                    `json:"nodeShape"`
	NodeShapeConfig   *NodeShapeConfig           `json:"nodeShapeConfig,omitempty"`
	NodeSourceDetails CreateNodeSourceDetails    `json:"nodeSourceDetails,omitempty"`
	SSHPublicKey      string                    `json:"sshPublicKey,omitempty"`
}

// NodePoolOptionsV3 contains the options available for specific fields that can be submitted
// to the CreateNodePool operation. Used by the
type NodePoolOptionsV3 struct {
	KubernetesVersions []string           `json:"kubernetesVersions"`
	Images             []string           `json:"images"`
	Shapes             []string           `json:"shapes"`
	Sources            []NodeSourceOption `json:"sources"`
}

// NodePoolNodeConfigDetails Contains the size and placement configuration of
// the node pool.
type NodePoolNodeConfigDetails struct {
	Size             uint32                           `json:"size"`
	NsgIds           []string                         `json:"nsgIds"`
	PlacementConfigs []NodePoolPlacementConfigDetails `json:"placementConfigs"`
}

// NodePoolPlacementConfigDetails contains the AD info of the subnet. A regional subnet
// spans all ADs in a region.
type NodePoolPlacementConfigDetails struct {
	AvailabilityDomain string `json:"availabilityDomain"`
	SubnetID           string `json:"subnetId"`
}

// ToProto converts a CreateNodePoolDetailsV3 to a NodePoolNewRequest object understood by grpc
func (v3 *CreateNodePoolDetailsV3) ToProto(enableNodePoolEnhancements bool) (*NewRequest, error) {
	var dst NewRequest
	if v3 != nil {
		dst.Name = v3.Name
		dst.CompartmentID = v3.CompartmentID
		dst.ClusterID = v3.ClusterID
		dst.K8SVersion = v3.KubernetesVersion
		dst.NodeImageName = v3.NodeImageName
		dst.NodeShape = v3.NodeShape
		if v3.NodeShapeConfig != nil && v3.NodeShapeConfig.Ocpus != nil {
			floatValue := protobuf.ToFloatValue(*v3.NodeShapeConfig.Ocpus)
			dst.NodeOcpus = &floatValue
		}
		if v3.NodeShapeConfig != nil && v3.NodeShapeConfig.MemoryInGBs != nil {
			floatValue := protobuf.ToFloatValue(*v3.NodeShapeConfig.MemoryInGBs)
			dst.NodeMemoryInGBs = &floatValue
		}

		if nodeSourceDetails, e := v3.NodeSourceDetails.UnmarshalPolymorphicJSON(v3.NodeSourceDetails.JsonData); e == nil {
			// UnmarshalPolymorphicJSON returns nil in the scenario where the node pool was created with a legacy
			// nodeImageId or nodeImageName, so don't throw an error
			if nodeSourceDetails != nil {
				if nodeSrcImgDetails, ok := nodeSourceDetails.(CreateNodeSourceViaImageDetails); ok {
					dst.NodeImageID = *nodeSrcImgDetails.ImageId
					if enableNodePoolEnhancements && nodeSrcImgDetails.BootVolumeSizeInGBs != nil {
						value := wrappers.UInt32Value{Value: *nodeSrcImgDetails.BootVolumeSizeInGBs}
						dst.NodeBootVolumeSizeInGBs = &value
					}
				} else {
					return nil, fmt.Errorf("unable to determine nodeSourceDetails type")
				}
			}
		} else {
			return nil, fmt.Errorf("unable to unmarshal nodeSourceDetails")
		}

		initialNodeLabels := KeyValuesToString(v3.InitialNodeLabels)
		dst.InitialNodeLabels = initialNodeLabels

		dst.SSHPublicKey = v3.SSHPublicKey
		dst.SubnetsInfo = make(map[string]*SubnetInfo)

		// nodeConfigDetails model.
		if len(v3.NodeConfigDetails.PlacementConfigs) > 0 {
			dst.Size = v3.NodeConfigDetails.Size
			for _, placementConfig := range v3.NodeConfigDetails.PlacementConfigs {
				dst.SubnetsInfo[CreateNodePoolSubnetInfoKey(placementConfig)] = &SubnetInfo{
					ID: placementConfig.SubnetID,
					AD: placementConfig.AvailabilityDomain,
				}
			}
		} else { // legacy model, use QuantityPerSubnet/subnetIds
			dst.QuantityPerSubnet = v3.QuantityPerSubnet
			for _, id := range v3.SubnetIDs {
				dst.SubnetsInfo[id] = &SubnetInfo{ID: id}
			}
		}
		dst.NsgIds = v3.NodeConfigDetails.NsgIds

		dst.NodeMetadata = make(map[string]string)
		for k, v := range v3.NodeMetadata {
			dst.NodeMetadata[k] = v
		}
	}

	return &dst, nil
}

// KeyValueV3 is used for holding a key/value pair whose value is a string
type KeyValueV3 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// String converts a key value pair to key=value pair
func (kv *KeyValueV3) String() string {
	return fmt.Sprintf("%s=%s", strings.TrimSpace(kv.Key), strings.TrimSpace(kv.Value))
}

// KeyValuesToString converts a slice of KeyValueV3 to a comma-separated string of key=value pairs
func KeyValuesToString(keyValues []KeyValueV3) string {
	kvs := []string{}
	for _, kv := range keyValues {
		if len(kv.Key) > 0 {
			kvs = append(kvs, kv.String())
		}
	}
	return strings.Join(kvs, ",")
}

// KeyValuesFromString creates a slice of KeyValueV3 from a comma-separated string of key=value pairs
func KeyValuesFromString(s string) []KeyValueV3 {
	keyValues := []KeyValueV3{}
	if len(s) > 0 {
		pairs := strings.Split(s, ",")
		for _, p := range pairs {
			parts := strings.Split(p, "=")
			kv := KeyValueV3{
				Key: parts[0],
			}
			if len(parts) > 1 {
				kv.Value = parts[1]
			}
			keyValues = append(keyValues, kv)
		}
	}
	return keyValues
}

// NodePoolCreateCLIResponseV3 is used by the CLI when creating a node pool
type NodePoolCreateCLIResponseV3 struct {
	WorkRequestID string `json:"workRequestId"`
}

// NodePoolDeleteCLIResponseV3 is used by the CLI when deleting a node pool
type NodePoolDeleteCLIResponseV3 struct {
	WorkRequestID string `json:"workRequestId"`
}

// NodePoolUpdateCLIResponseV3 is used by the CLI when updating a node pool
type NodePoolUpdateCLIResponseV3 struct {
	WorkRequestID string `json:"workRequestId"`
}

// helper functions
func convertSubnetInfoToUniqueSubnetIds(subnetsInfo map[string]*SubnetInfo) []string {
	subnetIds := make([]string, 0, len(subnetsInfo))
	seenSubnetIds := make(map[string]bool, len(subnetsInfo))
	for _, subnetInfo := range subnetsInfo {
		if _, ok := seenSubnetIds[subnetInfo.ID]; ok {
			continue
		}

		subnetIds = append(subnetIds, subnetInfo.ID)
		seenSubnetIds[subnetInfo.ID] = true
	}
	return subnetIds
}

func convertSubnetInfoToPlacementConfigs(subnetsInfo map[string]*SubnetInfo) []NodePoolPlacementConfigDetails {
	placementConfigs := make([]NodePoolPlacementConfigDetails, 0, len(subnetsInfo))
	for _, subnetInfo := range subnetsInfo {
		placementConfig := NodePoolPlacementConfigDetails{
			AvailabilityDomain: subnetInfo.GetAD(),
			SubnetID:           subnetInfo.GetID(),
		}
		placementConfigs = append(placementConfigs, placementConfig)
	}

	sort.SliceStable(placementConfigs, func(i, j int) bool {
		return placementConfigs[i].AvailabilityDomain < placementConfigs[j].AvailabilityDomain ||
			(placementConfigs[i].AvailabilityDomain == placementConfigs[j].AvailabilityDomain &&
				placementConfigs[i].SubnetID < placementConfigs[j].SubnetID)
	})

	return placementConfigs
}

func CreateNodePoolSubnetInfoKey(placementcfg NodePoolPlacementConfigDetails) string {
	keyPattern := "%s:%s"
	return fmt.Sprintf(keyPattern, placementcfg.AvailabilityDomain, placementcfg.SubnetID)
}
