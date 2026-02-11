locals {
  // Update the pop version corresponding to the pop build for app release
  pop_version = "a4a221a256a_212"

  // Update the ccm image sha value here for updating CCM versions for respective k8s versions across all realms
  ccm_default_mapping = {
    "default" = {
      "all" : {
        "default" : "oke-multiarch-1.23-526d1e6-171@sha256:85235e1fa24c41e5fb158346e3339fc680dcdce791735bfca25c7755a479e4c8",
        "v1.23" : "oke-multiarch-1.23-526d1e6-171@sha256:85235e1fa24c41e5fb158346e3339fc680dcdce791735bfca25c7755a479e4c8",
        "v1.24" : "v1.24-32be19ef595-4@sha256:3eda1610412ce5a3f6009b1d1a9219b3fdcc59009a8e3077a83f2b82142a586e",
        "v1.25" : "v1.25-d0b59914251-38@sha256:ffa44cb1e6bc5793859a1ebf7762bcd1731ec0dea426034c44bc77ffb12ece48",
        "v1.26" : "v1.26-8744a6c9ccd-42@sha256:bc20c825c5e3b5f40b56467e3b931597a6edef41cd0ab0cb20524b4cd8e603a0",
        "v1.27" : "v1.27-7e9bf7a9189-52@sha256:fd510337e52ef609ffb86953ef080b3f2fa1a19a0647f65dfbd2c1dfbda0df7a",
        "v1.28" : "v1.28-79d4f40b682-84@sha256:d8e0957a384781955a1e6b1dfb5498fa8ffb585face9a407486692ca685343fa",
        "v1.29" : "v1.29-393f7c992a6-112@sha256:cfc0512abe6e31a1e6020e45ebd7e79f89dc8f7bf5edc2655757762d5c662886",
        "v1.30" : "v1.30-8b278ca71ec-126@sha256:b52c65b13a3185413ddae10b0bb797d07b887ea893f6975e721962fcd6ddd3ed",
        "v1.31" : "v1.31-fe64ed05c1e-106@sha256:7b9ab84d2041d9f44022c49bc314348c4d30ff9c8c3a0544624a83113126ad2c",
        "v1.32": "v1.32-2900f005dfc-89@sha256:e51184fad8a9fbd76ea1822b4a39e9af1c2bda4f12b2a4f1d24f617c8afe8e4e",
        "v1.33": "v1.33-cb45871c215-59@sha256:f862ceccae104679949b6729d7d664435ac115327627c39520b73d63eb3ebb7f",
        "v1.34": "v1.34-4218f75593b-34@sha256:0cf1f22c21bcd5fda40024d3c4b8603af77f45612e6127192b241536939f95d7",
        "v1.35": "v1.34-4218f75593b-34@sha256:0cf1f22c21bcd5fda40024d3c4b8603af77f45612e6127192b241536939f95d7",
      }
    }
  }

  csi_default_mapping = {
    "default" = {
      "all" : {
        "default": "oke-multiarch-1.23-4c65264-146@sha256:defa64f16f0a16b84f3009cd6d902e4ce547dff68d3a5b85e510749d982de164",
        "v1.23" : "oke-multiarch-1.23-4c65264-146@sha256:defa64f16f0a16b84f3009cd6d902e4ce547dff68d3a5b85e510749d982de164",
        "v1.24" : "v1.24-32be19ef595-4@sha256:3eda1610412ce5a3f6009b1d1a9219b3fdcc59009a8e3077a83f2b82142a586e",
        "v1.25" : "v1.25-e402eabbe0a-25@sha256:af01ae775ba15f965f797ccea9aa6dece9e98136f42fb23c56775d7047307394",
        "v1.26" : "v1.26-d5c95b4f813-25@sha256:d401f00fc5f6d2710f916b44450c5f4264673f1403a4aabeba98393e318fcec1",
        "v1.27" : "v1.27-bff7091ad1d-51@sha256:10555f13db26cbcb35dcd684be88db9897e01ad25bdcd7d65e9cb4bd1c5f386f",
        "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
        "v1.29" : "v1.29-0f63a5020b8-110@sha256:466c7d32860ef68c4c98feba232409bf1580c732ee23fd3294ce59e2653bd125",
        "v1.30" : "v1.30-9696a00641f-5952@sha256:ef50fb8445e15b6e816bc68d6261f07e10e66b758da2923df023dbe4bd82da47",
        "v1.31" : "v1.31-4e30e28b828-80-csi@sha256:bf55f642531ebcb3e8ec09c3adeb9507552733df58e9fc6b7692bc241d5df2ad",
        "v1.32" : "v1.32-cdb98690a4c-88-csi@sha256:09e5e53cac2c153c47fac13016c8bff5453a59b4fd0febf78f149340df3d0a63",
        "v1.33" : "v1.33-fd3150dc2e1-58-csi@sha256:12e563da0eba8c4c6b3464b56d664de9c2a045291ec65b491382cc165e3b5c7b",
        "v1.34" : "v1.34-8913e88fdef-33-csi@sha256:1fbf5f93975972f8d11bbdf4337965fbe0c1bff5c177c2e2a732567ef32f75f9",
        "v1.35" : "v1.34-8913e88fdef-33-csi@sha256:1fbf5f93975972f8d11bbdf4337965fbe0c1bff5c177c2e2a732567ef32f75f9"
      }
    }
  }

  // Test any upstream version
  ccm_upstream_k8s_new_version_test = {
    "default" = {
      "all" : {
      }
    }
  }

  csi_upstream_k8s_new_version_test = {
    "default" = {
      "all" : {
      }
    }
  }

  global_default_values = {
    // The "default" mapping uses a SHA that references a manifest list. At runtime, the manifest list will resolve to
    // the actual image based on the node's architecture.


    // The "default" mapping uses a SHA that references a manifest list. At runtime, the manifest list will resolve to
    // the actual image based on the node's architecture.
    csi_fss_node_driver_registrar_image_version_mapping = {
      "default" = {
        "all" : "{}"
      }


      // To override property values for an environment declare a key
      // by environment name. To override values for a qualified realm, declare
      // a key with qualified realm name
    }

    // The "default" mapping uses a SHA that references a manifest list. At runtime, the manifest list will resolve to
    // the actual image based on the node's architecture.
    csi_fss_node_driver_image_version_mapping = {
      "default" = {
        "all" : "{}"
      }

      // To override property values for an environment declare a key
      // by environment name. To override values for a qualified realm, declare
      // a key with qualified realm name
    }

    csi-bv-expansion-enabled = {
      "default" = {
        "all" : "true"
      }
    }

    fss-csi-driver-enabled = {
      "default" = {
        "all" : "true"
      }
    }

    oci-service-controller-enabled = {
      "default" = {
        "all": "true"
      }
    }

    cpo-enable-resource-attribution = {
      "default" = {
        "all": "false"
      }
      "dev.oc1" = {
        "all": "true"
      }
      "integ.oc1" = {
        "all": "true"
      }
      "prd.oc1" = {
        "all": "true"
      }
      "prd.oc16" = {
        "all": "true"
      }
    }

    lustre-csi-driver-enabled = {
      "default" = {
        "all": "true"
      }
    }

    lustre-csi-controller-driver-enabled = {
      "default" = {
        "all": "true"
      }
    }

    lustre-csi-provisioner-worker-threads = {
      "default" = {
        "all": "15"
      }
    }


    // CCM related mappings
    ccm_image_version_mapping = {
      "default" = {
        "all" : jsonencode(local.ccm_default_mapping.default.all)
      }

      "dev.oc1" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
        "iad" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
        "phx" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
        "fra" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "integ.oc1" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
        "iad" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
        "phx" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          },
          local.ccm_upstream_k8s_new_version_test.default.all
        ))
        "fra" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }
      // To override property values for an environment declare a key
      // by environment name. To override values for a qualified realm, declare
      // a key with qualified realm name
      // https://developer.hashicorp.com/terraform/language/functions/merge
      "prd.oc1" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }


      "prd.oc8" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc9" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc10" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      // pinned mappings in oc43
      // hotfix applied for https://jira-sd.mc1.oracleiaas.com/browse/OKENP-36322
      "prd.oc43" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.23-526d1e6-171@sha256:85235e1fa24c41e5fb158346e3339fc680dcdce791735bfca25c7755a479e4c8",
            "v1.29" : "v1.29-0e75cc621e3-7019@sha256:3f8b9516f581eeeaafefcc2fbc448e09f44c1e7ad502761dc8334fadc167c809",
          }
        ))
      }

      // US - GOV realms
      "prd.oc2" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      "prd.oc3" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      "prd.oc23" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      // UK - GOV realms
      "prd.oc4" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-1.16-520cc1d-11@sha256:ba64f2c4ebf862d8e00d5c762e510bf41a7c9b735b5bb3a80a5ee33e344da7bf",
            "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }


      // ONSR realms
      "prd.oc5" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-1.16-520cc1d-11@sha256:7900589b191fb6a77b77172c3800428e4c435b54605cc033c8ebea4c60ae5df1",
            "v1.17" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22" : "oke-1.22-9893434-269@sha256:4d9775fe579b47b3e53c6a361c27114d55b97775d6040206f1b48d7f002684e6",
            "v1.23" : "oke-1.23-526d1e6-171@sha256:b738bf71baccc009e8eda5d818942a7d4c1be6d6dd63abe79c731aa2cafb036f",
            "v1.25" : "v1.25-756ad2ecbcd-31@sha256:9653868283b1b285daa6773383e0a77e5068dd67a3c32cd36b0a8f10d92ffbda",
            "v1.26" : "v1.26-046c0c74dc9-35@sha256:672d5c011956e4621b563e658c589dc2ae368d6a04dc04cc804765d8195bf029",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      "prd.oc6" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-1.16-520cc1d-11@sha256:7900589b191fb6a77b77172c3800428e4c435b54605cc033c8ebea4c60ae5df1",
            "v1.17" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22" : "oke-1.22-9f30fe0-220@sha256:fdbb7926235ecaa790b4d775ca7befe57f0fd8cd559afd6ead9a6ff765de8312",
            "v1.23" : "oke-1.23-b8a8dee-140@sha256:922eb0404d72f9abc63218ac4275b5676519af7698d770362a8eb9e4f3ba65a8",
            "v1.25" : "v1.25-756ad2ecbcd-31@sha256:9653868283b1b285daa6773383e0a77e5068dd67a3c32cd36b0a8f10d92ffbda",
            "v1.26" : "v1.26-046c0c74dc9-35@sha256:672d5c011956e4621b563e658c589dc2ae368d6a04dc04cc804765d8195bf029",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      "prd.oc11" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18" : "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21" : "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22" : "oke-1.22-9893434-269@sha256:4d9775fe579b47b3e53c6a361c27114d55b97775d6040206f1b48d7f002684e6",
            "v1.23" : "oke-1.23-526d1e6-171@sha256:b738bf71baccc009e8eda5d818942a7d4c1be6d6dd63abe79c731aa2cafb036f",
            "v1.25" : "v1.25-756ad2ecbcd-31@sha256:9653868283b1b285daa6773383e0a77e5068dd67a3c32cd36b0a8f10d92ffbda",
            "v1.26" : "v1.26-046c0c74dc9-35@sha256:672d5c011956e4621b563e658c589dc2ae368d6a04dc04cc804765d8195bf029",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
            "v1.31" : "v1.31-36db5994afd-108@sha256:9ea3a6aad84d3ff6ff7e54f4a6ef265a056cfde5c6c5bf3af2b2de907439db3d",
          }
        ))
      }

      "prd.oc14" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc16" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc17" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc19" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc20" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc22" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc24" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
          }
        ))
      }

      "prd.oc23" = {
        "all" : jsonencode(merge(local.ccm_default_mapping.default.all,
          {
            "default" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.28" : "v1.28-cb1635cc6c7-80@sha256:d9e51c8c78b3ef040e5739d6ac079d6587bb4b03d8b65579a877b4ec84c4f219",
          }
        ))
      }
    }

    // CSI Image Mappings
    csi_image_version_mapping = {
      "default" = {
        "all" : jsonencode(local.csi_default_mapping.default.all)
      }
      // To override property values for an environment declare a key
      // by environment name. To override values for a qualified realm, declare
      // a key with qualified realm name
      "dev.oc1" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
        "iad" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
        "phx" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
        "fra" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
      }

      "integ.oc1" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"

          }
        ))
        "iad" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
        "phx" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689",
          },
          local.csi_upstream_k8s_new_version_test.default.all
        ))
        "fra" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689"
          }
        ))
      }

      "prd.oc1" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689",
          }
        ))
      }

      "prd.oc8" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031"
          }
        ))
      }

      "prd.oc9" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031"
          }
        ))
      }

      "prd.oc10" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031"
          }
        ))
      }

      // US-GOV Realms
      "prd.oc2" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc3" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.20": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.21": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      // UK - GOV Realms
      "prd.oc4" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-1.16-520cc1d-11@sha256:ba64f2c4ebf862d8e00d5c762e510bf41a7c9b735b5bb3a80a5ee33e344da7bf",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      // ONSR Realms
      "prd.oc5" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-1.16-520cc1d-11@sha256:7900589b191fb6a77b77172c3800428e4c435b54605cc033c8ebea4c60ae5df1",
            "v1.17": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22": "oke-1.22-aa9948e-192@sha256:6407276d0df64c6b29e9beafe84bfe55191cb3ef153424a41d969bc1c0b0de52",
            "v1.23": "oke-1.23-16022dc-95@sha256:51de94592812ef652c880cd7ebfd0ddf1187d79e5fecf263253dfe72a72e8f38",
          }
        ))
      }

      "prd.oc6" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-1.16-520cc1d-11@sha256:7900589b191fb6a77b77172c3800428e4c435b54605cc033c8ebea4c60ae5df1",
            "v1.17": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22": "oke-1.22-9f30fe0-220@sha256:fdbb7926235ecaa790b4d775ca7befe57f0fd8cd559afd6ead9a6ff765de8312",
            "v1.23": "oke-1.23-b8a8dee-140@sha256:922eb0404d72f9abc63218ac4275b5676519af7698d770362a8eb9e4f3ba65a8",
          }
        ))
      }

      "prd.oc11" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.18": "oke-1.17-40e9a7a-13@sha256:424470ceb13e4b76f0a07e645bf2b37838f9a30709fdfdbce51b79748a6ff364",
            "v1.19": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.20": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.21": "oke-1.19-64ab664-255@sha256:fb4144e0480f120c54ea4688f70ef834b9a1fb0c03ffe0f03eca5afe61ae6765",
            "v1.22": "oke-1.22-aa9948e-192@sha256:6407276d0df64c6b29e9beafe84bfe55191cb3ef153424a41d969bc1c0b0de52",
            "v1.23": "oke-1.23-16022dc-95@sha256:51de94592812ef652c880cd7ebfd0ddf1187d79e5fecf263253dfe72a72e8f38",
          }
        ))
      }

      "prd.oc14" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc16" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc17" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc19" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc20" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
            "v1.22": "oke-multiarch-1.22-aa9948e-192@sha256:5bf9fef2c99f2c77fdb8027d8690900c99adef873bde9bcf6cd30db39ce467eb",
            "v1.23": "oke-multiarch-1.23-16022dc-95@sha256:2782b14bc755aa839ca68a86591007cf4dfb2c8419be14a5eae94b24a77f1031",
          }
        ))
      }

      "prd.oc22" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689",
          }
        ))
      }

      "prd.oc24" = {
        "all" : jsonencode(merge(local.csi_default_mapping.default.all,
          {
            "default": "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
            "v1.17": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.18": "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
            "v1.19": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.20": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.21": "oke-multiarch-1.19-73d694a-238@sha256:7215c4dcaeae4f199e82939a8a3dc73e519b00d0ac3714350b22002f8ae4f7aa",
            "v1.22": "oke-multiarch-1.22-d0bafe8-232@sha256:4697113594971e55df52d9e72dda7c381b431ec11d8d4622d82c7fafeb6c2689",
          }
        ))
      }
    }
  }
  global_default_values_by_property = {
  for property_name, property_value in local.global_default_values : property_name => merge(
    lookup(property_value, "default", {}),
    lookup(property_value, var.env, {}),
    lookup(property_value, "${var.env}.${var.realm}", {})
  )
  }

  global_default_values_list = flatten([
  for property_name, property_value in local.global_default_values_by_property : [
    {
      ad     = lookup(property_value, "ad", "all")
      group  = var.spectre_group_name
      name   = replace(property_name, "_", "-")
      region = var.execution_target.additional_locals.limits_region
      value  = lookup(property_value, var.execution_target.additional_locals.limits_region, lookup(property_value, "all", ""))
      min    = lookup(property_value, "min", null)
      max    = lookup(property_value, "max", null)
    }
  ] if length(property_value) > 0
  ])

  global_default_values_map = {
  for property in local.global_default_values_list : "${property.group}/${property.name}/${property.region}/${property.ad}" => property
  }
}

output "pop_version" {
  value = local.pop_version
}
