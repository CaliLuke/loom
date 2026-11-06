#!/usr/bin/env bash
# Object openness has one semantic owner: expr.AttributeExpr.ObjectOpenness.
# Framework code must not read or write the authored metadata key directly.
# The OpenAPI importer renderer is allowed because it emits DSL source text.
# Test files, testdata and integration_tests hold authored designs, which use
# the key as public DSL.
set -euo pipefail

allowed=(
  expr/object_openness.go
  internal/openapiimport/render_schema.go
)

status=0
while IFS= read -r match; do
  file=${match%%:*}
  skip=0
  for path in "${allowed[@]}"; do
    if [[ "$file" == "$path" ]]; then
      skip=1
    fi
  done
  if [[ "$skip" == 1 ]]; then
    continue
  fi
  echo "$match" >&2
  status=1
done < <(grep -R -n -E --include='*.go' --exclude='*_test.go' \
  --exclude-dir=testdata --exclude-dir=integration_tests --exclude-dir=.claude --exclude-dir=.git \
  '["`]openapi:additionalProperties["`]' . | sed 's|^\./||' | grep -v -E '^[^:]+:[0-9]+:[[:space:]]*//' || true)

if [[ "$status" != 0 ]]; then
  echo "use expr.AttributeExpr.ObjectOpenness or CloseObject instead of the openapi:additionalProperties metadata key" >&2
fi
exit "$status"
