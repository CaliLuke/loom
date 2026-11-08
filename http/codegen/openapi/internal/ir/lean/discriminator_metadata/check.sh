#!/bin/sh
set -eu

model_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$model_dir"

lean_version=$(lean --version)
case "$lean_version" in
	"Lean (version 4.35.0,"*) ;;
	*)
		echo "expected Lean 4.35.0, got: $lean_version" >&2
		exit 1
		;;
esac

while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	grep -Fq "theorem $theorem :" DiscriminatorMetadata.lean || {
		echo "required theorem is missing: $theorem" >&2
		exit 1
	}
done < required-theorems.txt

audit=$(lean -DwarningAsError=true DiscriminatorMetadata.lean)
printf '%s\n' "$audit"
while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	printf '%s\n' "$audit" | grep -Fq "'Loom.OpenAPI.DiscriminatorMetadata.$theorem' does not depend on any axioms" || {
		echo "required theorem axiom audit failed: $theorem" >&2
		exit 1
	}
done < required-theorems.txt

# DiscriminatorMapping (#1005) admits only Lean's standard axioms.
lean -DwarningAsError=true DiscriminatorMapping.lean
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
cp DiscriminatorMapping.lean "$scratch/Audit.lean"
while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	grep -Eq "^theorem $theorem( |$)" DiscriminatorMapping.lean || {
		echo "required theorem is missing: $theorem" >&2
		exit 1
	}
	printf '#print axioms Loom.OpenAPI.DiscriminatorMapping.%s\n' "$theorem" >> "$scratch/Audit.lean"
done < mapping-required-theorems.txt
mapping_audit=$(cd "$scratch" && lean Audit.lean | tr '\n' ' ' | sed "s/'Loom\./\n'Loom./g")
printf '%s\n' "$mapping_audit"
while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	line=$(printf '%s\n' "$mapping_audit" | grep -F "'Loom.OpenAPI.DiscriminatorMapping.$theorem'") || {
		echo "required theorem axiom audit failed: $theorem" >&2
		exit 1
	}
	axioms=$(printf '%s\n' "$line" | sed -n 's/.*depends on axioms: \[\(.*\)\].*/\1/p' | tr -d ' ' | tr ',' '\n')
	for axiom in $axioms; do
		case "$axiom" in
			propext|Classical.choice|Quot.sound) ;;
			*)
				echo "theorem $theorem depends on unexpected axiom: $axiom" >&2
				exit 1
				;;
		esac
	done
done < mapping-required-theorems.txt
