#!/bin/bash

set -e
set -o pipefail

exec &> >(tee -a "${ODO_APPLICATION_ROOT}/var/start.log")

echo "Starting release validation"

if [[ -z "$ODO_APPLICATION_ROOT" ]]; then
  echo "No ODO_APPLICATION_ROOT defined, cannot continue"
  exit 1
fi

JSON_FILE="${ODO_APPLICATION_ROOT}/image_versions.json"

COMPARTMENT_OCID=$STEWARD_TENANCY_OCID

export OCI_CLI_USE_INSTANCE_METADATA_SERVICE=true

if [ -n "$cpo_image_1" ]; then
  all_images=()

  for var in $(compgen -v cpo_image_); do
    all_images+=("${!var}")
  done

  repo_name="oke-public-cloud-provider-oci"

  existing_tags=$(oci artifacts container image list \
    --compartment-id "$COMPARTMENT_OCID" \
    --region "$REGION" \
    --repository-name "$repo_name" \
    --all \
    --auth instance_principal \
    --query 'data.items[*].[["version"], ["digest"]]' \
    --output json)

  declare -A existing_tags_map
  while IFS= read -r item; do
    image_tag=$(jq -r '.[0][0]' <<< "$item")
    digest=$(jq -r '.[1][0]' <<< "$item")
    existing_tags_map["$image_tag-$digest"]=true
  done < <(jq -c '.[]' <<< "$existing_tags")

  missing_tags=()

  for tag in "${all_images[@]}"; do
    image_tag=${tag%%@*}
    digest=${tag#*@}
    if [[ ! ${existing_tags_map["$image_tag-$digest"]} ]]; then
      missing_tags+=("$image_tag")
    fi
  done

  missing_tags_with_error=()

  for tag in "${missing_tags[@]}"; do
    if [[ $tag =~ oke-multiarch ]]; then
      echo "Warning: Missing image: $tag"
    elif [[ $tag =~ ^v([0-9]+)\.([0-9]+)- ]]; then
      major_version=${BASH_REMATCH[1]}
      minor_version=${BASH_REMATCH[2]}
      if [ "$major_version" -eq 1 ] && [ "$minor_version" -lt 29 ]; then
        echo "Warning: Missing image: $tag"
      else
        echo "Error: Missing image: $tag"
        missing_tags_with_error+=("$tag")
      fi
    else
      echo "Error: Unknown tag format: $tag"
      missing_tags_with_error+=("$tag")
    fi
  done

  if (( ${#missing_tags_with_error[@]} > 0 )); then
    exit 1
  else
    echo "All images greater than v1.28 are present in OCIR"
  fi
else

  declare -A repo_tags_map
  declare -a missing_tags

  fetch_repository_tags() {
    local repo_name=$1
    oci artifacts container image list \
      --compartment-id "$COMPARTMENT_OCID" \
      --region "$REGION" \
      --repository-name "$repo_name" \
      --all \
      --auth instance_principal \
      --query 'data.items[*]."display-name"' \
      --output json | jq -r '.[]' | awk -F':' '{print $2}'
  }

  expected_repos=$(mktemp)
  jq -r '.images[] | keys[]' "$JSON_FILE" | sort -u > "$expected_repos"

  while read -r repo_name; do
    repo_tags=$(fetch_repository_tags "$repo_name" | sort -u || true)
    repo_tags_map["$repo_name"]="$repo_tags"

    repo_tags="${repo_tags_map[$repo_name]}"

    expected_tags=$(jq -r --arg repo "$repo_name" '.images[][$repo] // empty' "$JSON_FILE")

    IFS=$'\n'
    for tag in $expected_tags; do
      if echo "$repo_tags" | grep -qx "$tag"; then
        continue
      else
        missing_tags+=("$repo_name:$tag")
      fi
    done
  done < "$expected_repos"

  rm "$expected_repos"

  if [ -n "${missing_tags[*]}" ]; then
    echo "The following images are not present in OCIR:"
    for missing_tag in "${missing_tags[@]}"; do
        echo "  $missing_tag"
    done
    exit 1
  else
    echo "All images found in OCIR."
  fi
fi