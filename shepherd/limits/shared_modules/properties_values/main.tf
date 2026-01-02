output "regional_properties_values_overrides" {
  value = local.regional_properties_values_overrides
}

module "all_values_validation" {
  source = "../validation"

  # Validate only image mapping properties
  for_each = { for k, v in local.all_values : k => v if length(regexall("image-version-mapping", v.name)) > 0 }
  image_mapping_values = lookup(each.value, "value", null)
}

module "overrides_validation" {
  source = "../validation"

  # Validate only image mapping properties
  for_each = { for k, v in local.tenancy_overrides_final : k => v if length(regexall("image-version-mapping", v.name)) > 0 }
  image_mapping_values = lookup(each.value, "value", null)
}

resource "property_value" "values" {
  for_each = { for k, v in local.all_values : k => v if(contains(keys(local.created_definitions),  v.name)) }

  group  = each.value.group
  name   = each.value.name
  region = each.value.region
  ad     = each.value.ad
  min    = lookup(each.value, "min", null)
  max    = lookup(each.value, "max", null)
  value  = lookup(each.value, "value", null)
}

resource "property_override" "overrides" {
  for_each = { for k, v in local.tenancy_overrides_final : k => v if contains(keys(local.created_definitions), v.name) }

  group  = each.value.group
  name   = each.value.name
  region = each.value.region
  ad     = each.value.ad
  tag    = each.value.tag
  min    = lookup(each.value, "min", null)
  max    = lookup(each.value, "max", null)
  value  = lookup(each.value, "value", null)
}
