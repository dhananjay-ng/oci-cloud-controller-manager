#!/bin/bash

##################################################################################################
# This template can be used to tweak the environment variables needed to run the E2E tests locally #
# Default behavior:
# Runs test on an existing cluster in dev0-iad

# To run the tests:
# 1. Change the FOCUS variable here to specify the subset of E2E tests to run
# 2. Set CLUSTER_KUBECONFIG and CLOUD_CONFIG if needed
# 3. run 'source existing-cluster-dev0-env-template.sh' to set the variables
# 4. run 'make run-ccm-e2e-tests-local`
##################################################################################################

# The test suites to run (can replace or add tags)
export FOCUS="\[lustre\]"

# This variable tells the test not to install oci cli and wipe out your .oci/config
export LOCAL_RUN=1
export TC_BUILD=0

# This allows you to use your existing cluster
export ENABLE_CREATE_CLUSTER=false

# Set path to kubeconfig of existing cluster if it does not exist in default path. Defaults to $HOME/.kube/config_*
export CLUSTER_KUBECONFIG_AMD=$HOME/.kube/config
export CLUSTER_KUBECONFIG_ARM=$HOME/.kube/config

# Set path to cloud_config of existing cluster if it does not exist in default path. Defaults to $HOME/cloudconfig_*
export CLOUD_CONFIG_AMD="$HOME/cloudconfig_integ_phx"
export CLOUD_CONFIG_ARM="$HOME/cloudconfig_integ_phx"

export CNI_TYPE=OCI_VCN_IP_NATIVE

export IMAGE_PULL_REPO="iad.ocir.io/okedev/e2e-tests/"
export ADLOCATION="zkJl:PHX-AD-2"


#KMS key for CMEK testing
export CMEK_KMS_KEY="ocid1.key.oc1.phx.a5pq4liyaafqw.abyhqljt6vom6gkskvfk573dl4tfyah3qcumnfswvmssw5llud4n6g7hypoq"

#NSG Network security group created in cluster's VCN
export NSG_OCIDS="ocid1.networksecuritygroup.oc1.phx.aaaaaaaaubnnphlndatb5ojkb2fmo42gjdprett5z2oy5mhihh6i4nc5cgpa,ocid1.networksecuritygroup.oc1.phx.aaaaaaaa34hh25ocryiykb7t6ytukzkmzigchbs2h2ezvfpzpwkizticbwwa"

#Reserved IP created in e2e test compartment
export RESERVED_IP="144.24.42.104"

#Architecture to run tests on
export ARCHITECTURE_AMD="AMD"
export ARCHITECTURE_ARM="ARM"

#Focus the tests : ARM, AMD or BOTH
export SCOPE="AMD"

#NSG Network security group created in cluster's VCN for backend management, this NSG will have to be attached to the nodes manually for tests to pass
export BACKEND_NSG_OCIDS="ocid1.networksecuritygroup.oc1.phx.aaaaaaaa5mcufaaxor246s6ija72dbb3ybqtc5akdiys6yccs65kwyad3zqq"
# For debugging the tests in existing cluster, do not turn it off by default.
# export DELETE_NAMESPACE=false


#export HTTP_PROXY=http://www-proxy-idc.in.oracle.com:80
#export HTTPS_PROXY=http://www-proxy-idc.in.oracle.com:80


# FSS volume handle
# format is FileSystemOCID:serverIP:path

export FSS_VOLUME_HANDLE="ocid1.filesystem.oc1.phx.aaaaaaaaaacql6qjobuhqllqojxwiotqnb4c2ylefuyqaaaa:10.0.0.2:/FileSystem-Fss-Dyn-E2e"
export FSS_VOLUME_HANDLE_ARM="ocid1.filesystem.oc1.phx.aaaaaaaaaacql6qlobuhqllqojxwiotqnb4c2ylefuyqaaaa:10.0.0.2:/FileSystem-Fss-Dyn-E2E-Arm"

export MNT_TARGET_ID="ocid1.mounttarget.oc1.iad.aaaaacvippy7r7n4nfqwillqojxwiotjmfsc2ylefuyqaaaa"
export MNT_TARGET_SUBNET_ID="ocid1.subnet.oc1.iad.aaaaaaaapntwxrvyrnawcmhviwxwmf2vjdsigg32577rkqwrjzihypehgsta"
export MNT_TARGET_COMPARTMENT_ID="ocid1.compartment.oc1..aaaaaaaayt3yoxoggojnw5ym3qzd3ayzsbrugdgrjwovkphovctgmyk7vb4q"
export FSS_VOLUME_HANDLE_IPV6="ocid1.filesystem.oc1.phx.aaaaaaaaaahoeurpobuhqllqojxwiotqnb4c2ylefuyqaaaa:[2603:c020:11:1500:9b92:580f:cbd:fbd5]:/FileSystem-Fss-Dyn-E2e-IPv6"

export LUSTRE_VOLUME_HANDLE=""
export LUSTRE_VOLUME_HANDLE_ARM=""
export LUSTRE_SUBNET_CIDR=""
export ENABLE_LUSTRE_TESTS=true
export LUSTRE_WORKER_NODE_IMAGE="ocid1.image.oc1.phx.aaaaaaaaa4h5frsda4fiqjr73raowfig6vudw5b5ryvdbferzwg2tuqcv5cq"
#export LUSTRE_KMS_KEY="ocid1.key.oc1.phx.efuxb3z7aaef6.abyhqljsncobtl4xyg4z6toswulfiqgide7azpr6rf43ijli4cx3uuqpsxlq"

export STATIC_SNAPSHOT_COMPARTMENT_ID=""
export ENABLE_PARALLEL_RUN=false

# Workload Identity Principal Feature only available for ENHANCED_CLUSTER
export CLUSTER_TYPE="ENHANCED_CLUSTER"

# For SKE node, node_info, node_lifecycle controller tests against PDE
# To setup PDE and point your localhost:25000 to the PDE CP API refer: Refer: https://bitbucket.oci.oraclecorp.com/projects/OKE/repos/oke-control-plane/browse/personal-environments/README.md
# export CE_ENDPOINT_OVERRIDE="http://localhost:25000"

# Ip family of cluster to create cluster as per required ip stack
export CLUSTER_IP_FAMILY="IPv4"
export NP_IMAGE_OS="Oracle-Linux-8"
export SKIP_CLUSTER_DELETION="true"


export MNT_TARGET_ID_IPV6=""
export MNT_TARGET_SUBNET_ID_IPV6=""
export MNT_TARGET_SUBNET_ID_DUAL_STACK=""
export OCI_K8SSUBNET_IPV6="ocid1.subnet.oc1.phx.aaaaaaaao3iiruscssdr73icasz7pqte7bdfppoieilkcsg74eafadjvhf5q"
export OCI_K8SSUBNET_DUAL_STACK="ocid1.subnet.oc1.phx.aaaaaaaaqivqxfg44dvgarlrmwtsat5w3oa2t3i35z7lben4yg5jua4nxpxa"
export OCI_NODESUBNET_IPV6="ocid1.subnet.oc1.phx.aaaaaaaa6umsjm6bszml73xzll4t2p4u76z5xyrwepee42wxmh4bhusgbzxa"
export OCI_NODESUBNET_DUAL_STACK="ocid1.subnet.oc1.phx.aaaaaaaassdb32w5fq4rkeo3gt2lw6fnhhg4gqrdlehlgpamn42b35hb4w3q"
export LBRGNSUBNET_IPV6="ocid1.subnet.oc1.phx.aaaaaaaaj54463vrghmqii2g4xlegichxiqgajjwsjhotkj2xl73ckbbaqca"
export LBRGNSUBNET_DUAL_STACK="ocid1.subnet.oc1.phx.aaaaaaaadbekwkbgmbe6fcfcgj23ibf2jtohissdn6flk3u7vykdovjr3jqa"
export EXISTING_CLUSTER_OCID="ocid1.clusterinteg.oc1.phx.aaaaaaaauaw7hmcndm4c7w7xpayi57rccofswp4kwvz6p5rmmcpczkzmkhwa"


export OKE_ENDPOINT=containerengine-integ.us-phoenix-1.oci.oraclecloud.com
export VCN="ocid1.vcn.oc1.phx.amaaaaaah4gjgpyawe7o7zyoiela3ctlakaixqim3vcjoc2hkjcqbqglgjyq"
export POD_SUBNET_AMD="ocid1.subnet.oc1.phx.aaaaaaaarg7i3skfu6aki6ue6d7wtbaj72arkqyzsjl7vw4pg4a2n6nvgorq"
export POD_SUBNET_ARM="ocid1.subnet.oc1.phx.aaaaaaaarg7i3skfu6aki6ue6d7wtbaj72arkqyzsjl7vw4pg4a2n6nvgorq"
export POD_SUBNET="ocid1.subnet.oc1.phx.aaaaaaaarg7i3skfu6aki6ue6d7wtbaj72arkqyzsjl7vw4pg4a2n6nvgorq"
export POD_SUBNET_DUAL_STACK="ocid1.subnet.oc1.phx.aaaaaaaav2utw4kk2rgilrseo5zbp7hwzcgnanokx2kdvsfjksj5e4gbrt2a"
export ADD_OKE_SYSTEM_TAGS=true

export CREATE_UHP_NODEPOOL=false
export NODE_SHAPE="VM.Standard.A1.Flex"
export NODE_SHAPE_ARM="VM.Standard.A1.Flex"
export NODE_SHAPE_AMD="VM.Standard2.1"
export LBRGNSUBNET="ocid1.subnet.oc1.phx.aaaaaaaadbekwkbgmbe6fcfcgj23ibf2jtohissdn6flk3u7vykdovjr3jqa"
export OCI_NODESUBNET="ocid1.subnet.oc1.phx.aaaaaaaaegz7agsaqt5d6fdbwhtq2m3vmlzwhiivnuyzyf3b3mz7y6b4qfya"

export NODEPOOL_SIZE="3"
export E2E_NODE_COUNT=1
export E2E_BRANCH=private-e2e
export COMPARTMENT="ocid1.compartment.oc1..aaaaaaaa7cgk4a3z2vknlyixzhibvdpauuvxdbs5c6n67yktnxdgh53zhibq"