package metrics

const (
	// LBProvision is the OCI metric suffix for LB provision
	LBProvision = "LB_PROVISION"
	// LBUpdate is the OCI metric suffix for LB update
	LBUpdate = "LB_UPDATE"
	// LBDelete is the OCI metric suffix for LB delete
	LBDelete = "LB_DELETE"
	// LBPodReadinessSync is the OCI metric suffix for LB pod readiness sync
	LBPodReadinessSync = "LB_PODREADINESS_SYNC"

	// NLBProvision is the OCI metric suffix for NLB provision
	NLBProvision = "NLB_PROVISION"
	// NLBUpdate is the OCI metric suffix for NLB update
	NLBUpdate = "NLB_UPDATE"
	// NLBDelete is the OCI metric suffix for NLB delete
	NLBDelete = "NLB_DELETE"
	// NLBPodReadinessSync is the OCI metric suffix for NLB pod readiness sync
	NLBPodReadinessSync = "NLB_PODREADINESS_SYNC"

	// NSGProvision is the OCI metric suffix for NLB provision
	NSGProvision = "NSG_PROVISION"
	// NSGUpdate is the OCI metric suffix for NLB update
	NSGUpdate = "NSG_UPDATE"
	// NSGDelete is the OCI metric suffix for NLB delete
	NSGDelete = "NSG_DELETE"

	// PVProvision is the OCI metric suffix for PV provision
	PVProvision = "PV_PROVISION"
	// PVAttach is the OCI metric suffix for PV attach
	PVAttach = "PV_ATTACH"
	// PVAttachRWX is the OCI metric suffix for PV attach RWX
	PVAttachRWX = "PV_ATTACH_RWX"
	// PVDetach is the OCI metric suffix for PV detach
	PVDetach = "PV_DETACH"
	// PVDetachRWX is the OCI metric suffix for PV detach RWX
	PVDetachRWX = "PV_DETACH_RWX"
	// PVDelete is the OCI metric suffix for PV delete
	PVDelete = "PV_DELETE"
	// PVExpand is the OCI metric suffix for PV Expand
	PVExpand = "PV_EXPAND"
	// PVClone is the OCI metric for PV Clone
	PVClone = "PV_CLONE"
	// PVUpdate is the OCI metric for PV Update
	PVUpdate = "PV_UPDATE"

	// FSSProvision is the OCI metric suffix for FSS provision
	FSSProvision = "FSS_PROVISION"
	// FSSDelete is the OCI metric suffix for FSS delete
	FSSDelete = "FSS_DELETE"
	// FSSUpdate is the OCI metric suffix for FSS update
	FSSUpdate = "FSS_UPDATE"

	// MTProvision is the OCI metric suffix for Mount Target provision
	MTProvision = "MT_PROVISION"
	// MTDelete is the OCI metric suffix for Mount Target delete
	MTDelete = "MT_DELETE"

	// ExportProvision is the OCI metric suffix for Export provision
	ExportProvision = "EXP_PROVISION"
	// ExportDelete is the OCI metric suffix for Export delete
	ExportDelete = "EXP_DELETE"

	// BlockSnapshotProvision is the OCI metric suffix for Block Volume Snapshot Provision
	BlockSnapshotProvision = "BSNAP_PROVISION"
	// BlockSnapshotDelete is the OCI metric suffix for Block Volume Snapshot Delete
	BlockSnapshotDelete = "BSNAP_DELETE"
	// BlockSnapshotRestore is the OCI metric suffix for Block Volume Snapshot Restore
	BlockSnapshotRestore = "BSNAP_RESTORE"

	// FssAllProvision is the OCI metric suffix for FSS end to end provision
	FssAllProvision = "FSS_ALL_PROVISION"

	// FssAllDelete is the OCI metric suffix for FSS end to end deletion
	FssAllDelete = "FSS_ALL_DELETE"

	// LustreProvision is the OCI metric suffix for Lustre end to end provision
	LustreProvision = "LUSTRE_PROVISION"
	// LustreDelete is the OCI metric suffix for Lustre end to end deletion
	LustreDelete = "LUSTRE_DELETE"

	ResourceOCIDDimension     = "resourceOCID"
	ComponentDimension        = "component"
	BackendSetsCountDimension = "backendSetsCount"
	VolumeVpusPerGBDimension  = "vpusPerGB"
	InstanceIdDimension       = "instanceID"
	ClusterOCID               = "clusterId"
)
