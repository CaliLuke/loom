#!/bin/sh
set -eu

model_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$model_dir"

lean_version=$(lean --version)
case "$lean_version" in
	"Lean (version 4.34.1,"*) ;;
	*)
		echo "expected Lean 4.34.1, got: $lean_version" >&2
		exit 1
		;;
esac

while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	grep -Fq "theorem $theorem :" DocumentIdentity.lean || {
		echo "required theorem is missing: $theorem" >&2
		exit 1
	}
done < required-theorems.txt

audit=$(lean -DwarningAsError=true DocumentIdentity.lean)
printf '%s\n' "$audit"
while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	printf '%s\n' "$audit" | grep -Fq "'Loom.OpenAPI.DocumentIdentity.$theorem' does not depend on any axioms" || {
		echo "required theorem axiom audit failed: $theorem" >&2
		exit 1
	}
done < required-theorems.txt
