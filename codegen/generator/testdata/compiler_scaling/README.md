# Compilation scaling probe

These are reproducible source inputs for the compiler ownership investigation.
The .go.txt suffix keeps this external-module example out of repository package
discovery; the copy commands below give each source its compiling filename. They live
in testdata and do not add a default benchmark or integration gate. Copy the two
Go sources into separate packages in a disposable module; do not run generation
inside the framework checkout.

The design independently varies endpoint count, unique type count, record width
and recursive edges. The probe runs DSL evaluation and generation in one fresh
process, records cumulative allocation and GC/heap counters, then writes an
allocation profile. It intentionally has no timing assertion and forces no GC.

## Reproduction

Use Go 1.27.3 and the framework's pinned dependencies. Set LOOM_REPO to an absolute
checkout path and LOOM_REVISION to the exact pushed commit under examination.
For comparison, create separate disposable modules for each source revision.

```sh
probe_root=$(mktemp -d)
mkdir -p "$probe_root/design" "$probe_root/probe"
cp "$LOOM_REPO/codegen/generator/testdata/compiler_scaling/design.go.txt" "$probe_root/design/design.go"
cp "$LOOM_REPO/codegen/generator/testdata/compiler_scaling/probe.go.txt" "$probe_root/probe/main.go"
cd "$probe_root"
go mod init example.com/scaling
go mod edit -go=1.27.3
go get "github.com/CaliLuke/loom@$LOOM_REVISION"
go mod tidy
go build -o compile-probe ./probe
GOMAXPROCS=10 SCALING_METHODS=16 SCALING_TYPES=1 SCALING_WIDTH=32 SCALING_RECURSIVE=0 \
  /usr/bin/time -l ./compile-probe allocs.pprof
go test ./gen/...
```

For an unpushed local experiment, explicitly replace the framework module with
its absolute source checkout before tidy/build. Record the exact source diff;
a local replacement is not a reproducible remote revision. The `time -l` command
above is for macOS. Its peak RSS and wall/CPU time include profile writing after
the probe's MemStats sample; compare like-for-like runs. Cumulative allocation is
allocated bytes, not simultaneous live memory.

Use a fresh output directory when changing workload shape so old generated files
do not contaminate comparison. Compare every relative artifact path and byte,
including the manifest, across independent processes with the same source identity.
Remote revision pins derive different loom_version metadata in the manifest; enumerate those expected metadata differences when comparing two
remote-pin runs. The recorded exact-byte experiments instead used local source
replacements for both versions, giving both the same `(devel)` version metadata.
To reproduce that comparison, export each exact pushed revision into a separate
source directory and use that directory as its module's local replacement. Record
the pushed revision and confirm the exports have no source edits; this keeps the
source reproducible while equalizing only build identity. Do not discard arbitrary
generated differences as metadata. Compilation
of unchanged output under the same runtime/dependency inputs can reuse previous
compilation evidence. Do not clear shared Go caches or conflate executable build
cost with generator execution. Alternate source order for paired timings, keep
other task builds out of the measurement, and report medians and limitations.

## Workloads

| Case | Methods | Unique types | Width | Recursive | Artifacts |
|---|---:|---:|---:|---|---:|
| Small | 4 | 1 | 8 | No | 16 |
| Reused | 16 | 1 | 8 | No | 20 |
| Unique | 16 | 16 | 8 | No | 20 |
| Wide | 16 | 1 | 32 | No | 20 |
| Recursive | 16 | 1 | 8 | Yes | 21 |

Unset parameters use the source's defaults. Positive method/type/width integers
are required. SCALING_RECURSIVE=1 adds a recursive record edge. These fixtures
sample independent dimensions; they are not a proof of global asymptotic bounds.
The direct field-query, normalization-depth and fingerprint-depth regressions
exercise specific previously multiplicative work with deterministic allocation
checks and recorded semantic equivalence.

## Recorded evidence

See [the compilation architecture](../../../COMPILATION.md) for stage-by-stage
contracts, measurements and caveats. The final comparison uses the shared-method
semantic stage f88a5d32d as its baseline. Final source tree and medians are
recorded there alongside verification and measurement limits. The private real-design
probe is not copied into this repository; these synthetic inputs are public
reproduction evidence for scaling behavior, not a substitute for that workload.

Full CLI measurement is a separate experiment: it includes package discovery,
helper compilation/linking, generator execution and staged installation. Record
whether the Go caches are warm. For warm comparisons, first run each exact
source version once without timing it, then alternate the measured pairs. If
measuring simultaneous process-tree memory,
sample the CLI and its descendants and identify the sampling interval/overhead;
single-process maximum RSS is not a process-tree maximum. The recorded CLI
experiment samples `ps` every 50 ms plus command overhead, so its peak is sampled,
not an exact maximum. CLI/probe command headers differ intentionally; comparisons
between two CLI runs require no normalization.
