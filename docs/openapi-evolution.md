# Check OpenAPI Evolution with oasdiff

Use `oasdiff` to compare a committed Loom OpenAPI contract with a regenerated
revision. This workflow reports compatibility changes before they reach API
consumers.

Loom does not include a diff engine. The consuming repository owns its oasdiff
version, policy, severity overrides, and ignores.

## Distinguish schema identity from automatic reuse

Loom shares automatic request-body, parameter and header definitions only when
their contents, including examples, agree. Response components compare the HTTP
status and complete response shape, except for the top-level description. Operations can
share a data schema while retaining different examples in their own definitions.
Changing an example can therefore remove an automatic component and replace its
references with inline definitions. These automatic references are not stable
identifiers. Explicitly authored component names and schema names retain their
existing contracts.

For both OpenAPI 3.1 and 3.2, operation-specific response descriptions appear
beside `$ref`. The shared component has a neutral status description:

```yaml
responses:
  "400":
    $ref: '#/components/responses/BadRequestError'
    description: |-
      invalid_account: Account is invalid.

      invalid_metadata: Metadata is invalid.
```

Changing only that description does not create another component or change its
automatic name. Different schemas, headers, media types, links, summaries,
extensions or examples still prevent sharing. Explicit component names remain
separate. An explicitly omitted 3.2 description retains its omission policy and
3.1 fallback text. Unnamed single-use responses can remain inline.

Regenerate OpenAPI after upgrading. Tools that read response descriptions must
apply Reference Object description overrides, including an explicit empty string,
instead of reading only `components.responses`. This is standard
[OpenAPI 3.1 Reference Object behavior](https://spec.openapis.org/oas/v3.1.1.html#reference-object),
also supported by OpenAPI 3.2. Existing automatic component names and hash
suffixes can change during this migration; there is no legacy allocation mode.

Review both the OpenAPI diff and regenerated SDK and validator output when
updating Loom. Moving an equivalent definition inline preserves its contract,
but a client generator may produce different declarations or exports. Schema
equivalence alone does not prove compatibility of the generated client API.

## Select the OpenAPI compatibility target

Loom emits OpenAPI 3.2 by default. oasdiff `v1.33.0` reads 3.2 documents, but
its documentation says that 3.2 coverage remains under development.

Use Loom's 3.1 compatibility target for this conservative baseline:

```go
var _ = API("MyAPI", func() {
    Meta("openapi:version", "3.1")
    Meta("openapi:output", "json")
})
```

OpenAPI 3.1 support is generally available in oasdiff. Review the
[oasdiff OpenAPI support guide](https://github.com/oasdiff/oasdiff/blob/main/docs/OPENAPI-31.md)
before changing this target.

## Pin oasdiff

This guide uses these tested versions:

- oasdiff CLI `v1.33.0`
- oasdiff action `v0.1.18`, commit
  `b9325c9e0a27ab65b0da3b766522cedec6be81dc`

Install the pinned CLI without adding it to the service module:

```bash
go install github.com/oasdiff/oasdiff@v1.33.0
```

Pin the action to its full commit SHA in CI. Keep the release tag in a comment
so reviewers can identify the version.

For an upgrade:

1. Read the oasdiff CLI and action release notes.
2. Update both pins in one change.
3. Review OpenAPI version support and changed checks.
4. Run the normal comparison and the known-breaking canary below.
5. Review every policy or ignore-file change separately.

## Run the local gate

Replace the design package if your module uses another path.

```bash
DESIGN_PACKAGE=example.com/myservice/design
OPENAPI_SPEC=gen/http/openapi.json

go tool loom gen "$DESIGN_PACKAGE"
git diff --exit-code -- gen/http/openapi.json gen/http/openapi.yaml

oasdiff breaking \
  --fail-on ERR \
  --format githubactions \
  --allow-external-refs=false \
  -- \
  "origin/main:${OPENAPI_SPEC}" \
  "$OPENAPI_SPEC"
```

Run the command from the repository that contains the generated specification.
Fetch `origin/main` first when the local Git object is missing.

`--fail-on ERR` blocks definite breaking changes. oasdiff still reports `WARN`
findings for review. Use `--fail-on WARN` when project policy blocks both
levels.

Create a non-blocking Markdown changelog for review:

```bash
oasdiff changelog \
  --format markdown \
  --allow-external-refs=false \
  -- \
  "origin/main:${OPENAPI_SPEC}" \
  "$OPENAPI_SPEC" \
  > openapi-changelog.md
```

The changelog command does not set a blocking threshold in this recipe.

### Test a known breaking change

Run this canary when adopting or upgrading oasdiff:

1. Create a temporary branch.
2. Remove one published success response from the Loom design.
3. Regenerate the OpenAPI contract.
4. Run the local `oasdiff breaking` command above.
5. Make sure the command exits with status `1` and reports an `ERR` finding.
6. Revert the temporary design change and regenerate.

This canary proves that the selected version and project policy block a known
breaking change.

## Use the CI example

Copy [the pinned GitHub Actions example](examples/oasdiff.yml) to
`.github/workflows/oasdiff.yml`. Replace `DESIGN_PACKAGE` when required.

The example performs these tasks in order:

1. Fetch the pull request's base branch.
2. Regenerate the Loom contract.
3. Fail when committed JSON or YAML output is stale.
4. Add a non-blocking Markdown changelog to the job summary.
5. Report `WARN` findings and block `ERR` findings.

The revision input reads the regenerated working-tree file. The base input uses
the committed file from the pull request's base Git revision.

## Keep policy consumer-owned

Store project policy in the consuming repository. Common files include:

- `.oasdiff.yaml`
- a severity override file
- a narrow `ERR` ignore file
- a narrow `WARN` ignore file

Loom does not generate or overwrite these files. Give each ignore an owner,
reason, and removal condition. Prefer a severity override when project policy
classifies a check differently.

Start with `ERR` as the blocking threshold and review every `WARN`. A project
can later block `WARN` after it has removed noisy or ambiguous findings.

## Protect untrusted CI

The CLI resolves external `$ref` values by default. Keep
`--allow-external-refs=false` for untrusted pull requests.

The oasdiff action defaults this setting to `false`. The example sets it
explicitly. Enable external references only after defining safe file and
network loading rules.

The action also enables encrypted hosted review by default. It uploads an
encrypted comparison to oasdiff.com and can add a review link to the pull
request.

Set `review: false` when no specification data can leave CI. The example uses
this setting. For the CLI, omit `--open` to keep review data local.

## Understand the coverage boundary

| Contract area | Required coverage |
|---|---|
| Published standard OpenAPI fields | oasdiff breaking gate and changelog |
| Generated transport responses | Loom response-contract scaffold |
| `x-loom-*` extension behavior | Loom-owned focused tests |
| Business behavior and fixtures | Application-owned tests |

oasdiff can report text changes inside extensions. It cannot prove the runtime
meaning of Loom-specific extensions such as `x-loom-async`.

Use both layers. oasdiff protects the published standard contract, while Loom
scenarios prove that the implementation produces each declared response.
