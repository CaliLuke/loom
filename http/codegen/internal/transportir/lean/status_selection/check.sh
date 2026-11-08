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
	grep -Fq "theorem $theorem " StatusSelection.lean || {
		echo "required theorem is missing: $theorem" >&2
		exit 1
	}
done < required-theorems.txt

audit=$(lean -DwarningAsError=true StatusSelection.lean)
printf '%s\n' "$audit"
while IFS= read -r theorem; do
	[ -n "$theorem" ] || continue
	line=$(printf '%s\n' "$audit" | grep -F "'Loom.HTTP.StatusSelection.$theorem' " || true)
	case "$line" in
		*"does not depend on any axioms"*) ;;
		*"depends on axioms: ["*)
			axioms=${line#*\[}
			axioms=${axioms%\]*}
			for axiom in $(printf '%s' "$axioms" | tr ',' ' '); do
				case "$axiom" in
					propext|Quot.sound|Classical.choice) ;;
					*)
						echo "required theorem uses a non-standard axiom: $theorem: $axiom" >&2
						exit 1
						;;
				esac
			done
			;;
		*)
			echo "required theorem axiom audit failed: $theorem" >&2
			exit 1
			;;
	esac
done < required-theorems.txt
