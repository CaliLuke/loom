#!/usr/bin/env bash
# Shared discovery of hand-written Go files, sourced by lint_gofmt.sh and
# fmt.sh. It covers root-module packages, testdata packages and the nested
# integration-test modules.
#
# Generated gen/ trees and the integration fixture modules under
# */integration_tests/fixtures/ are excluded: `make generated-code-quality`
# covers them.

# hand_written_go_files prints the hand-written Go files under the current
# directory, one per line, sorted.
hand_written_go_files() {
  find . \( -path ./.git -o -type d -name gen -o -path '*/integration_tests/fixtures' \
    -o \( -type d ! -path . -exec test -e '{}/.git' \; \) \) -prune -o \
    -type f -name '*.go' -print |
    LC_ALL=C sort
}

# load_hand_written_go_files stores the discovered files in the global array
# GO_FILES and fails when discovery finds none.
load_hand_written_go_files() {
  GO_FILES=()
  local file
  while IFS= read -r file; do
    GO_FILES+=("$file")
  done < <(hand_written_go_files)
  if [ "${#GO_FILES[@]}" -eq 0 ]; then
    echo "Go source discovery found no hand-written Go files" >&2
    return 1
  fi
}
