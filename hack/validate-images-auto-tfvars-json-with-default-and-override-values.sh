#!/bin/bash
set -euo pipefail
dnf install -y jq

echo "Validating default_values.tf and locals.tf image mappings against MANIFEST.csv"

manifest="MANIFEST.csv"
defaults_tf="./shepherd/limits/shared_modules/properties_values/default_values.tf"
overrides_tf="./shepherd/limits/shared_modules/properties_values/locals.tf"
all_values="/tmp/all_values.json"

for f in "$manifest" "$defaults_tf" "$overrides_tf"; do
  [[ -f $f ]] || { >&2 echo "File not found: $f"; exit 1; }
done

(source ./shepherd/limits/scripts/gen_images_tfvars.sh MANIFEST.csv) > $all_values

echo "Generated all images in the manifest : "
cat $all_values

manifest_images=$(
  grep -oE '"name": *"[^"]+"' "$all_values" |
  sed -E 's/.*"name": *"//' |
  sed -E 's/"//' |
  sed -E 's/_DOT_/./g' |
  sed -E 's/^oke-[^_]+__//' |
  sort -u
)

extract_images_from_tf() {
  local tf_file=$1
  grep -oE '[^"[:space:]]+@sha256:[A-Fa-f0-9]+' "$tf_file" | \
  sed -E 's/@sha256:[A-Fa-f0-9]+//' | \
  sort -u
}

default_images=$(extract_images_from_tf "$defaults_tf")
override_images=$(extract_images_from_tf "$overrides_tf")
all_tf_images=$(printf "%s\n%s\n" "$default_images" "$override_images" | sort -u)

missing_in_manifest=()
for tf_img in $all_tf_images; do
  if ! grep -qF "$tf_img" <<< "$manifest_images"; then
    missing_in_manifest+=("$tf_img")
  fi
done

if [ ${#missing_in_manifest[@]} -eq 0 ]; then
  echo "SUCCESS: All images from default and overrides image mappings are present in MANIFEST.csv"
else
  echo "ERROR: The following images were not found in MANIFEST.csv:"
  printf '> %s\n' "${missing_in_manifest[@]}"
  exit 1
fi