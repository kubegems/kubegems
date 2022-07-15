#!/usr/bin/env bash
# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

noLicenseFiles=$(
    find . -type f -iname '*.go' ! -path '*/vendor/*' -exec sh -c 'head -n3 $1 | grep -Eq "(Copyright|generated|GENERATED)" || echo $1' {} {} \;
)

for path in $noLicenseFiles; do
echo -e "$(cat hack/boilerplate.go.txt)\n" | cat - $path > temp && mv temp $path
done
