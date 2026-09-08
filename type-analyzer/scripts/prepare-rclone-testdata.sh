#!/usr/bin/env sh
set -eu

# Builds the two opt-in integration-test fixtures described in README.md.
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
testdata_dir="$project_dir/testdata"
module_dir="$testdata_dir/rclone-module"
vendor_dir="$testdata_dir/rclone-vendor"
rclone_url="https://github.com/rclone/rclone.git"
rclone_ref="v1.68.2"

if [ -e "$module_dir" ] || [ -e "$vendor_dir" ]; then
  echo "rclone fixtures already exist; remove $module_dir and $vendor_dir before rebuilding" >&2
  exit 1
fi

git clone --depth 1 --branch "$rclone_ref" "$rclone_url" "$module_dir"
cp -a "$module_dir" "$vendor_dir"
(
  cd "$vendor_dir"
  go mod vendor
  rm go.mod go.sum
)

echo "created $module_dir and $vendor_dir"
