locals {
  tally_csi_overrides = {
    "v1.31" : "v1.31-d02f08265e8-67@sha256:4d0b75d5875307be05859d157e9d647b0703f878100e49a6a0df38b53d2dcaa3",
    "v1.32" : "v1.32-bf20a443646-33@sha256:6ca6fd685627f329769e110f34c4d48d78621859539ce64aa2de4fa868ed10a0"
  }

  karpenter_la_ccm_overrides_v1-17-1 = {
    "default" : "oke-multiarch-1.16-520cc1d-11@sha256:5a38b559cbb0a027b06f9381973974854b7bc5c5085ddd9e225ddf02820cdc78",
    "v1.17" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
    "v1.18" : "oke-multiarch-1.17-40e9a7a-13@sha256:60b1e805918f93e14bf618df8e224d8ac6de004496cf484c1ffd6bc74d1e38d9",
    "v1.19" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
    "v1.20" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
    "v1.21" : "oke-multiarch-1.19-64ab664-255@sha256:c0b0b665735d3288d0f8991c792c51aa00f9aaa031e2ffdd5ecca0238c03f28b",
    "v1.22" : "oke-multiarch-1.22-9893434-269@sha256:ceba7b8788c84d494113c862cd03dce2cc2c7b52c451ebeaa6eee88a97a4d8db",
    "v1.32" : "v1.32-8567820c2b2-7061@sha256:294c15051daea44308c17674f40434eb81dc9c7b6e0ff3af1505a2be39a5caf0",
    "v1.33" : "v1.33-227c27250c9-7060@sha256:f4288aaa8d38737a4c91e83f3ad35f8e517158bf8e5a395a640026f3f581f2a8",
    "v1.34" : "v1.34-219abcce797-7062@sha256:dbdfe17351eadbea3e9a8d9daf5cf55c6341217b2a16c9654cb2c0c9cc66a506",
    "v1.35" : "v1.34-219abcce797-7062@sha256:dbdfe17351eadbea3e9a8d9daf5cf55c6341217b2a16c9654cb2c0c9cc66a506"
  }

  // !!! Do not play with fire, only used by ci-cd pipeline
  oci_cnp_dev_override = {
  }

  // !!! Do not play with fire, only used by ci-cd pipeline
  oci_cnp_dev_ccm_override = {
  }
  // !!! Do not play with fire, only used by ci-cd pipeline
  oci_cnp_dev_csi_override = {
  }

  tenancy_property_overrides = {
    "oc1" = {
      "ccm-image-version-mapping" = {
        overrides = [
          /*
            oci-cnp-dev tenancy override. Not to be removed as it is used by pipelines
          */
          {
            regions      = ["phx"]
            env          = "integ"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.oci_cnp_dev_ccm_override))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaajol5woa4is3merb234fy4b46bps2nsjr3lcz7rvgj25dr5dxfmnq"
          },
          /*
          pin karpenter LA customer with CCM version v1.17.1
          pre-Flex CIDR changes. The overrides will be removed after the customer acts on the CN
          <CN link>
          */

          // karpenter LA overrides begins

          // IAD, PHX and GRU
          {
            regions      = ["iad", "phx", "gru"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaouhzxquloatd265invfv6faq34qqb76wgthnjxerddolkmyrgbnq"
          },

          // FRA, IAD
          {
            regions      = ["iad", "fra"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaawxdcmyu3bxuemm3yfj7jojapxsm6dmyx6s344bn6zsqb2ebznoyq"
          },

          // LHR, FRA
          {
            regions      = ["lhr", "fra"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa4wptnxymnypvjjltnejidchjhz6uimlhru7rdi5qb6qlnmrtgu3a"
          },

          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaacfk25l2yzn7ebafcpmobdukcgdr3frvhp6brtin65ypm7obv35cq"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaamfhabpfqsnuwc5643f6mr7yrnf6krsqjwxsaab5s4hhbpvral7sq"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaz3zvzfb4afndqhyljo75yoat5azx6yw5hppwxqv3felvg3flex4q"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaawllqtz5xfodqtftcpdc26amqb2jyvnfjnn3h5h5dmyc5lril3dea"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaau2ps7wkdgghrhqetihs2veiqfvyopqjv43xyfyoxqxdeob6yq2na"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaasrbi2t6cokzboloyeewnox4buqkaejs47n7nnbr2i4rczosus6vq"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa5eerf7ajzwvn4p4h7has7dg7fziwmnyyorhqo3xv3ob5lpxqczea"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa5ps76wflywhom3i4gvoltpxqxx25so4npbpiguykkfoqaj223fwq"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa5trur7whdyytam4nmh3tinrx2yfqnbss6yzz4q6i7gmm2leagnkq"
          },
          // IAD
          {
            regions      = ["iad"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaougcn7snh444oh4r5n6wfsfirzng2zwlnn6endtbllaiebfcbmma"
          },

          // PHX
          {
            regions      = ["phx"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaizwb7xbe7nt3wfb2jdrnsehrbip53s64qtwi2hx4y3ydkdmaeywq"
          },
          // PHX
          {
            regions      = ["phx"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaajol5woa4is3merb234fy4b46bps2nsjr3lcz7rvgj25dr5dxfmnq"
          },


          // HYD
          {
            regions      = ["hyd"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaiuttuudiwzzab44lf2xvhaqv3yecttzgyyh75h3uvhbow73xjbiq"
          },
          // HYD
          {
            regions      = ["hyd"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaayn6s77e3y4hgz56uzwvay3jrpekafo6ycr5pr5xdsg3gahcygx7a"
          },


          // LHR
          {
            regions      = ["lhr"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaufx55y3yism5pgygpqrdqcuzj3sghxqcgkhwgpt4ucaloqqi2laa"
          },
          // LHR
          {
            regions      = ["lhr"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa4wptnxymnypvjjltnejidchjhz6uimlhru7rdi5qb6qlnmrtgu3a"
          },

          // BOM
          {
            regions      = ["bom"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaagaa3pjerxezczu42udm7be5pbybsss5riojedzzuxi3wh73qdlla"
          },

          // YNY
          {
            regions      = ["yny"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa6ma7kq3bsif76uzqidv22cajs3fpesgpqmmsgxihlbcemkklrsqa"
          },


          // YUL
          {
            regions      = ["yul"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaau6tqqxwoo36xtiqtqgdfnaepwwdcgwcldg6stlrbxju4jzs76k2q"
          },

          // YYZ
          {
            regions      = ["yyz"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaazphi6fix3jx6syklx5qdnc4bsdwci7jubbqgg4j673n4zq6xpv4a"
          },


          // RUH
          {
            regions      = ["ruh"]
            env          = "prd"
            value        = jsonencode(merge(local.ccm_default_mapping.default.all, local.karpenter_la_ccm_overrides_v1-17-1))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaamlwq2br5i2t2o5guzkxkzuedfxlw4ifrxfdnz6yb5nzrs5jaz7hq"
          }
          // karpenter LA overrides ends
        ]
      },
      "csi-image-version-mapping" = {
        overrides = [
          /*
            oci-cnp-dev tenancy override. Not to be removed as it is used by pipelines
          */
          {
            regions      = ["phx"]
            env          = "integ"
            value        = jsonencode(merge(local.csi_default_mapping.default.all, local.oci_cnp_dev_csi_override))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaajol5woa4is3merb234fy4b46bps2nsjr3lcz7rvgj25dr5dxfmnq"
          },
          /*
            Tally Prod CSI Pin Override
          */
          {
            regions      = ["bom"]
            env          = "prd"
            value        = jsonencode(merge(local.csi_default_mapping.default.all, local.tally_csi_overrides))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaosp7wej2fqrvacexolafmu3kw3euc2k5lwuirycvwzhspoleqrsq"
          },
          /*
            Tally QA CSI Pin Override
          */
          {
            regions      = ["bom"]
            env          = "prd"
            value        = jsonencode(merge(local.csi_default_mapping.default.all, local.tally_csi_overrides))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaxtpwu4jog5czcjmawmwptkuqjfimnc57iij6asca4xwjx3wkf3pq"
          },
          /*
            Tally DEV CSI Pin Override
          */
          {
            regions      = ["bom"]
            env          = "prd"
            value        = jsonencode(merge(local.csi_default_mapping.default.all, local.tally_csi_overrides))
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaacihe4hnurwedhwfbgvexk2uemvwpyoq37vywgl6yh5n3a7wpopca"
          },
        ]
      },
      "lustre-csi-driver-enabled" = {
        /* hilbert tenancy - Temple*/
        overrides = [
          {
            regions      = ["aga"]
            env          = "prd"
            value        = "false"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaad6ghxkqwame6d4ioz2ejmrhksjxb7pif3fpp3r44fweejt2jvptq"
          }
        ]
      },
      "lustre-csi-controller-driver-enabled" = {
        /* hilbert tenancy - Temple*/
        overrides = [
          {
            regions      = ["aga"]
            env          = "prd"
            value        = "false"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaad6ghxkqwame6d4ioz2ejmrhksjxb7pif3fpp3r44fweejt2jvptq"
          }
        ]
      },
      "oci-service-controller-enabled" = {
        overrides = [
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaanrbjdkpfvz2w2ve67ecxy24pa6k3teofpvlbviyl2r5rsbwseqvq"
          },
          {
            regions      = ["phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa7wzb7yxrfd3vyhwbkgt5yzgbp6nqmf2xegio5ssi6lvxfwps4w3a"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaahp7giiq4smoqb5hmqnz5xmplfwuyfpovxivy22qczlrlgkzns7qq"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaahm2iczdm7ekujxrwgeicw6k2fedhsl742gauuvpxxbz6uvhlcekq"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaapagptfoo7j4ccd3j3vv3fbz3kwiz5jxcgmemxvt2fikluz6w3gqa"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa5uvuf7b3bmylrw725q4drkef5ftbeuoqfvqptzam655d44ncv5mq"
          },
          {
            regions      = ["phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaelop2bappx2pqqxzqv4myxtexgzuvwceexk3xi4zs44mk2tvvswa"
          },
          {
            regions      = ["iad"]
            env          = "dev"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaalbb2c4nf3ptr356lej2rpkjzy5cbcatbwqsphoseqokcjtueunha"
          },
          {
            regions      = ["iad"]
            env          = "integ"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaabg3uraonkvuh6kvczem6zuk6bwpvz4rca5jmhiqxzlcb4dhzxyqq"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaafoai2hnvga3lzgy3ljsz4n6vlcl6wdgec25k3ogjqg43yloxwkoa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaauklway4qjfxmzffkzloi65uxf2r5ak4rp7cccxlq6krk4seww7pa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaakjncznwynlcsjpxrqub3jskbzmz3qlkgoffiv7yjmyrfqqgy7gaq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaapbxhbccxcjrb4wnlyaix44kkcl2tgtj2lyrlcqcctqmmi26yehxa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaogcmvxwqogkitjiepz3kqdv2irjuyhcrcmp7yinh3ytca3v56aza"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa5husazs2ytt4vmvztlaifts4zraa26vz6p7i76vgugoic7a5ddga"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaooqw2ki26evdbangxtahcgcbqr7ycygxkxr5y5d6vd5xwl7eqj2a"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaapofprzxfp4munko4mvqc3u4nqegtkru6uzbaut774xrgmvsco2qa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaao5xkfdkgtgakm2gngfktpoxddhgak3sqlbuuc6ipsj4uqijhavsa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaalgye6or5foisxcezh5hrrloiqedji3gseyz4jlkjy4ixa7en4yra"
          },
          {
            regions      = ["iad", "phx", "fra", "yyz"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa62vpswgwthtfeiqsapy5dydvyk2bzzq6v3txo6gvh5g6o65ljuha"
          },
          {
            regions      = ["fra"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaafipe4lmow7rfrn5f3egpg3xgur6v2q2wgvb3id4ehwujnpu5mb5q"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaahzy3x4boh7ipxyft2rowu2xeglvanlfewudbnueugsieyuojkldq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaazo2fdm3n4e5ldbcuw4nqbszce4wzynnff3vswas3frbdznidf73a"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaafyd4yolgubij6qz7kgxxl5ps2qhqkfoqrvxtjvdccy66ifxmglia"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaj4ymkytn4cwasyktbc5rbqiald2enkoarepdv7pawyuw2tkazisa"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa3iy7zjbsehobgcdr3hr54ohhfzz6gb3mlrx3nil3rc5mefibhdqq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaageglfk6dntbw6xlw3a2zwyxmojqdltjrrk6qrhergy2mmnuvgzga"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa6myvkd2ri4u2p5rdgydenosfxgjrorlgxzede2dlmq37cbdagffq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaiirvljo5b5ps4etbubykf3sorn46ih2cq47zbjwqpj3x3jmcxara"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaal4apn7aq4e3h4zv6cqz56omhwgmnrd6sepnjc3x2vifhdaes4oq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaqeltzy3b3ioqnyqusa2po52d3ygn24yb72b44ceau4jdwoy2pnzq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa64d4rod2cgrqleyskgao4wkbeosylqellmeqbvxxkybhonlrbgk"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa4fdy3gbpphhp2v7lrworctpyjli5yvwhoa7db6uutr4xqwquq5eq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaasyc4pviq2l4mjafhhk2hejvsprv52jblh53jtujqoxi4knifzgya"
          },
          {
            regions      = ["iad", "phx", "fra", "yyz"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaatf553tfabldbvae4sfe3gqizjetiitaa6y7zyx2pazejqrufxlya"
          },
          {
            regions      = ["iad"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaagwykkho6dzrdst3nur52t3z5p6qk77z7dibjrnkbsfqm4peycqra"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaavyaz2rpyk5f54eqs4izohq2go3t54l437njmig2aonxqwboyqo3a"
          },
          {
            regions      = ["iad", "phx", "fra"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaa4wptnxymnypvjjltnejidchjhz6uimlhru7rdi5qb6qlnmrtgu3a"
          },
          {
            regions      = ["iad", "phx", "fra", "yyz"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaypz3jeouf67rycnobkfaj7zepotuvpoqtqipfai3qs4qw7ok6yna"
          },
          {
            regions      = ["iad", "phx", "fra", "yyz"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaymp6s54butpavvjb6fcwq3pzrrqbb33v7sh6qvzdqkimnbuqasla"
          },
          {
            regions      = ["phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaajznex5attydtrmrgudwayqu7kn4krasw2ct4h4pwz7nwbfxoyd4q"
          },
          {
            regions      = ["phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaax7tm7jtfarexna447cmubjxwou6lug42jss2ddyis63wqo3lrpda"
          },
          {
            regions      = ["iad", "phx"]
            env          = "dev"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaizwb7xbe7nt3wfb2jdrnsehrbip53s64qtwi2hx4y3ydkdmaeywq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaat37ab62ltpvzgyoydasbfig3gcmccxwzvbi6yoh6ewqiiswps6sq"
          },
          {
            regions      = ["iad", "phx"]
            env          = "prd"
            value        = "true"
            tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaaft2mucrx352mvjgie7ez7lobrnvn56hv5lkxm7kbv6m6wrwo22q"
          }
        ]
      }
    },
    "oc16" = {
      "ccm-image-version-mapping" = {
        overrides = [
          // Hurray, no snowflakes
        ]
      }
    }
  }
}
#Usage:
/* "oc1" = {
     "property-name" = {
        overrides = [
           {
             regions = ["region"]
             env = "environment"
             value = "value"
             tenancy_ocid = "ocid1.tenancy.ocx..aaaaaaaaaaaaaaaaaaaaaaa"
           }
        ]
     }
  }
*/
