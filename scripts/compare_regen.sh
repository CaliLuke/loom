#!/usr/bin/env bash

set -euo pipefail

usage() {
	cat <<'EOF'
Usage: compare_regen.sh [options]

Regenerate a Goa application twice:
1. With the application's pinned Goa dependency.
2. With the local goa-light checkout replacing goa.design/goa/v3.

Then write the generated trees, logs, unified diff, and a compact summary to an
output directory.

Options:
  --target-repo PATH      Target application repository. Required.
  --design-package PKG    Design package to generate. Default: <module>/design
  --output-dir PATH       Directory for artifacts. Default:
                          /tmp/goa-regen-compare-<repo>-<timestamp>
  --goa-root PATH         Local goa-light checkout. Default: current repo root
  --goa-ai-root PATH      Local goa-ai checkout to replace goa.design/goa-ai.
                          Default: none
  --baseline-label NAME   Label for pinned dependency output. Default: baseline
  --candidate-label NAME  Label for local goa-light output. Default: candidate
  --keep-temp             Keep the temporary modfile directory.
  --help                  Show this help.
EOF
}

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

abs_path() {
	local path="$1"
	if [[ -d "$path" ]]; then
		(
			cd "$path"
			pwd
		)
		return
	fi
	local dir base
	dir="$(dirname "$path")"
	base="$(basename "$path")"
	(
		cd "$dir"
		printf '%s/%s\n' "$(pwd)" "$base"
	)
}

repo_name_from_path() {
	basename "$1" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-'
}

target_repo=""
goa_root="$(pwd)"
goa_ai_root=""
output_dir=""
design_package=""
baseline_label="baseline"
candidate_label="candidate"
keep_temp=0

while [[ $# -gt 0 ]]; do
	case "$1" in
		--target-repo)
			target_repo="${2:?missing value for --target-repo}"
			shift 2
			;;
		--design-package)
			design_package="${2:?missing value for --design-package}"
			shift 2
			;;
		--output-dir)
			output_dir="${2:?missing value for --output-dir}"
			shift 2
			;;
		--goa-root)
			goa_root="${2:?missing value for --goa-root}"
			shift 2
			;;
		--goa-ai-root)
			goa_ai_root="${2:?missing value for --goa-ai-root}"
			shift 2
			;;
		--baseline-label)
			baseline_label="${2:?missing value for --baseline-label}"
			shift 2
			;;
		--candidate-label)
			candidate_label="${2:?missing value for --candidate-label}"
			shift 2
			;;
		--keep-temp)
			keep_temp=1
			shift
			;;
		--help|-h)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1"
			;;
	esac
done

[[ -n "$target_repo" ]] || die "--target-repo is required"
target_repo="$(abs_path "$target_repo")"
goa_root="$(abs_path "$goa_root")"
if [[ -n "$goa_ai_root" ]]; then
	goa_ai_root="$(abs_path "$goa_ai_root")"
fi

[[ -d "$target_repo" ]] || die "target repo does not exist: $target_repo"
[[ -f "$target_repo/go.mod" ]] || die "target repo is missing go.mod: $target_repo"
[[ -f "$goa_root/go.mod" ]] || die "goa root is missing go.mod: $goa_root"
if [[ -n "$goa_ai_root" ]]; then
	[[ -f "$goa_ai_root/go.mod" ]] || die "goa-ai root is missing go.mod: $goa_ai_root"
fi

target_module="$(cd "$target_repo" && go list -m -f '{{.Path}}')"
if [[ -z "$design_package" ]]; then
	design_package="${target_module}/design"
fi

if [[ -z "$output_dir" ]]; then
	timestamp="$(date +%Y%m%d-%H%M%S)"
	output_dir="/tmp/goa-regen-compare-$(repo_name_from_path "$target_repo")-${timestamp}"
fi
output_dir="$(abs_path "$output_dir")"

baseline_root="${output_dir}/${baseline_label}"
candidate_root="${output_dir}/${candidate_label}"
baseline_gen="${baseline_root}/gen"
candidate_gen="${candidate_root}/gen"
baseline_log="${output_dir}/${baseline_label}.log"
candidate_log="${output_dir}/${candidate_label}.log"
diff_patch="${output_dir}/diff.patch"
summary_file="${output_dir}/summary.txt"
commands_file="${output_dir}/commands.txt"

mkdir -p "$baseline_root" "$candidate_root"

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goa-regen-compare.XXXXXX")"
cleanup() {
	if [[ "$keep_temp" -eq 0 ]]; then
		rm -rf "$temp_dir"
	else
		printf 'kept temp dir: %s\n' "$temp_dir"
	fi
}
trap cleanup EXIT

cp "$target_repo/go.mod" "$temp_dir/go.mod"
if [[ -f "$target_repo/go.sum" ]]; then
	cp "$target_repo/go.sum" "$temp_dir/go.sum"
fi
(
	cd "$temp_dir"
	go mod edit -replace "goa.design/goa/v3=${goa_root}"
	if [[ -n "$goa_ai_root" ]]; then
		go mod edit -replace "goa.design/goa-ai=${goa_ai_root}"
	fi
)

cat >"$commands_file" <<EOF
target_repo=$target_repo
design_package=$design_package
goa_root=$goa_root
goa_ai_root=$goa_ai_root
baseline_command=(cd $target_repo && env GOWORK=off GOFLAGS=-mod=mod go run goa.design/goa/v3/cmd/goa gen $design_package -o $baseline_root)
candidate_command=(cd $target_repo && env GOWORK=off GOFLAGS=-mod=mod\ -modfile=$temp_dir/go.mod go run goa.design/goa/v3/cmd/goa gen $design_package -o $candidate_root)
EOF

printf 'Generating pinned baseline into %s\n' "$baseline_root"
(
	cd "$target_repo"
	env GOWORK=off GOFLAGS='-mod=mod' \
		go run goa.design/goa/v3/cmd/goa gen "$design_package" -o "$baseline_root"
) >"$baseline_log" 2>&1

printf 'Generating local candidate into %s\n' "$candidate_root"
(
	cd "$target_repo"
	env GOWORK=off GOFLAGS="-mod=mod -modfile=$temp_dir/go.mod" \
		go run goa.design/goa/v3/cmd/goa gen "$design_package" -o "$candidate_root"
) >"$candidate_log" 2>&1

printf 'Writing diff artifacts into %s\n' "$output_dir"
diff -ruN "$baseline_gen" "$candidate_gen" >"$diff_patch" || true

python3 - "$baseline_gen" "$candidate_gen" "$summary_file" <<'PY'
import collections
import filecmp
import os
import sys

baseline = sys.argv[1]
candidate = sys.argv[2]
summary_path = sys.argv[3]

changed = []
left_only = []
right_only = []

def walk(rel):
    left = os.path.join(baseline, rel)
    right = os.path.join(candidate, rel)
    cmp = filecmp.dircmp(left, right)
    left_only.extend(os.path.join(rel, name).lstrip("./") for name in sorted(cmp.left_only))
    right_only.extend(os.path.join(rel, name).lstrip("./") for name in sorted(cmp.right_only))
    for name in sorted(cmp.diff_files):
        changed.append(os.path.join(rel, name).lstrip("./"))
    for name in sorted(cmp.common_dirs):
        walk(os.path.join(rel, name))

walk(".")

all_changed = sorted(set(changed + left_only + right_only))
bucket_counts = collections.Counter()
for rel in all_changed:
    top = rel.split(os.sep, 1)[0] if rel else "."
    bucket_counts[top] += 1

with open(summary_path, "w", encoding="utf-8") as fh:
    fh.write(f"baseline={baseline}\n")
    fh.write(f"candidate={candidate}\n")
    fh.write(f"changed_entries={len(all_changed)}\n")
    fh.write(f"differing_files={len(changed)}\n")
    fh.write(f"only_in_baseline={len(left_only)}\n")
    fh.write(f"only_in_candidate={len(right_only)}\n\n")
    fh.write("Top-level buckets:\n")
    for bucket, count in sorted(bucket_counts.items(), key=lambda item: (-item[1], item[0])):
        fh.write(f"  {bucket}: {count}\n")

    def write_section(title, items):
        fh.write(f"\n{title} ({len(items)}):\n")
        limit = 50
        for item in items[:limit]:
            fh.write(f"  {item}\n")
        if len(items) > limit:
            fh.write(f"  ... {len(items) - limit} more\n")

    write_section("Differing files", changed)
    write_section("Only in baseline", left_only)
    write_section("Only in candidate", right_only)
PY

printf '\nArtifacts:\n'
printf '  baseline gen: %s\n' "$baseline_gen"
printf '  candidate gen: %s\n' "$candidate_gen"
printf '  summary: %s\n' "$summary_file"
printf '  diff: %s\n' "$diff_patch"
printf '  baseline log: %s\n' "$baseline_log"
printf '  candidate log: %s\n' "$candidate_log"
printf '  commands: %s\n' "$commands_file"
printf '\nSummary preview:\n'
sed -n '1,20p' "$summary_file"
