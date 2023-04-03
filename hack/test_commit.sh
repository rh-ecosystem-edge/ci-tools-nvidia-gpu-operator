#! /usr/bin/env bash

# Script was forked from https://github.com/rh-ecosystem-edge/ci-artifacts/blob/master/testing/test-commit.sh

set -o pipefail
set -o errexit
set -o nounset

echo "===> Runnin test_command.sh"


THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
BASE_DIR="${THIS_DIR}/../"
cd "$BASE_DIR"

ANCHOR="test_command: "

commit="HEAD"
parent=$(git log --pretty=%P -n 1 $commit)

if grep -q " " <<< "$parent"; then
    commit=$(cut -d" " -f2 <<< "$parent")
    echo "===> HEAD is a merge commit. Taking the 2nd parent from $parent"
else
    echo "===> HEAD is a simple commit."
fi

git show --quiet "$commit"

echo ""

testpaths=$(git log --format=%B -n 1 $commit | { grep -i "$ANCHOR" || true ;} | cut -b$(echo "$ANCHOR" | wc -c)-)

if [[ -z "$testpaths" ]]; then
    echo "Nothing to test in $commit."
    exit 1
fi


while read cmd;
do
    echo "Running (make target) test_commit: $cmd"
    echo
    make ${cmd}
    echo ""
done <<< "$testpaths"

echo "All done."
