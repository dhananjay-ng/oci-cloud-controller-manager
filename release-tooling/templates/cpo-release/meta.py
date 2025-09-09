# constants
integ_phx_target = "integ.oc1.us-phoenix-1.cell0"
prod_phx_target = "prd.oc1.us-phoenix-1.cell0"

app     = meta_variables.get('app')
infra   = meta_variables.get('infra')

# load common variables

with open("templates/meta-commons.py") as commons:
    exec(commons.read())

config_id="a89d8954-ce10-48c3-8825-3de6716117c8"

app_release_template = {
    "alias": "image-push",
    "flock_name": "oke-ccm-csi",
    "type": "Application"
}

infra_release_template = {
    "alias": "mapping-update",
    "flock_name": "oke-ccm-csi",
    "type": "Infrastructure"
}

meta_template = {
    "template_name": "cpo-release",
    "release_template_location": "../../releases/cpo-release/",
    "disclaimer": "",
    "components": [],
    "flocks": [],
    "release_templates": [],
    "schedule": oke_common_release_schedule,
    "exclude_locations": excluded_locations
}

if app:
    meta_template["release_templates"].append(app_release_template)
    meta_template["flocks"].append({
        "shepherd_project_name": "oke",
        "flock_name": "oke-ccm-csi",
        "new_deployment_scope": {
            "config_id_or_commit_id": config_id,
        },
        "deployment_scope_type": "regional",
        "current_prod_deployment_scope": {
            "execution_target": prod_phx_target,
            "change_type": "Application"
        },
        "logical_phases": [
            {
                "scope": "region",
                "name": "main",
                "shepherd_phase": lambda scope: f"prd.{scope.realm.lower()}",
                "execution_target": lambda scope: f"prd.{scope.realm.lower()}.{scope.region_name.lower()}.cell0"
            }
        ]
    })
    meta_template["components"].append({
        "flock": "oke-ccm-csi",
        "artifact_version_resolvers": [
            {
                "resolver_type": "static",
                "artifacts": {
                    "oke-public-cloud-provider-oci__v1_DOT_28-79d4f40b682-84",
                    "oke-public-cloud-provider-oci__v1_DOT_29-393f7c992a6-112",
                    "oke-public-cloud-provider-oci__v1_DOT_30-b38c1eb59d9-122",
                    "oke-public-cloud-provider-oci__v1_DOT_31-ae10afe1fb6-88",
                    "oke-public-cloud-provider-oci__v1_DOT_32-3fd404a8f60-58",
                    "oke-public-cloud-provider-oci__v1_DOT_33-46dbdee2d2b-28",
                    "release-validator-ccm-csi"
                },
                "resolver_params": {
                    "static_versions": {
                        "oke-public-cloud-provider-oci__v1_DOT_28-79d4f40b682-84": {
                            "version": "v1.28-79d4f40b682-84",
                            "summary": "CPO image to be pushed for release"
                        },
                        "oke-public-cloud-provider-oci__v1_DOT_29-393f7c992a6-112": {
                            "version": "v1.29-393f7c992a6-112",
                            "summary": "CPO image to be pushed for release"
                        },
                        "oke-public-cloud-provider-oci__v1_DOT_30-b38c1eb59d9-122": {
                            "version": "v1.30-b38c1eb59d9-122",
                            "summary": "CPO image to be pushed for release"
                        },
                        "oke-public-cloud-provider-oci__v1_DOT_31-ae10afe1fb6-88": {
                            "version": "v1.31-ae10afe1fb6-88",
                            "summary": "CPO image to be pushed for release"
                        },
                        "oke-public-cloud-provider-oci__v1_DOT_32-3fd404a8f60-58": {
                            "version": "v1.32-3fd404a8f60-58",
                            "summary": "CPO image to be pushed for release"
                        },
                        "oke-public-cloud-provider-oci__v1_DOT_33-46dbdee2d2b-28": {
                            "version": "v1.33-46dbdee2d2b-28",
                            "summary": "CPO image to be pushed for release"
                        },
                        "release-validator-ccm-csi": {
                            "version": "dd05bb6acaf_124",
                            "summary": "Release validator POP image to be pushed for release"
                        },
                    }
                }
            }
        ]
    })

if infra:
    meta_template["release_templates"].append(infra_release_template)
    meta_template["flocks"].append({
        "shepherd_project_name": "oke",
        "flock_name": "oke-ccm-csi",
        "new_deployment_scope": {
            "config_id_or_commit_id": config_id,
        },
        "deployment_scope_type": "regional",
        "current_prod_deployment_scope": {
            "execution_target": prod_phx_target,
            "change_type": "Infrastructure"
        },
        "logical_phases": [
            {
                "scope": "region",
                "name": "main",
                "shepherd_phase": lambda scope: f"prd.{scope.realm.lower()}",
                "execution_target": lambda scope: f"prd.{scope.realm.lower()}.{scope.region_name.lower()}.cell0"
            },
            {
                "scope": "region",
                "name": "main",
                "shepherd_phase": lambda scope: f"prd.{scope.realm.lower()}",
                "execution_target": lambda scope: f"spectre.values.prd.{scope.realm.lower()}.{scope.region_name.lower()}"
            }
        ]
    })
