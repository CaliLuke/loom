# Full-corpus audit source

This directory preserves the additional orchestration used during the value
contract migration. The reusable revision-comparison framework is in the
[parent directory](../README.md); the exported-design compile corpus is in
[`internal/testdatacompile`](../../testdatacompile/README.md).

The files under `testdata/session-source` preserve the original audit scripts,
Go overlays and supporting inputs. The inventory records original paths and
SHA-256 hashes. Go overlays use `.go.txt` filenames because they are historical
source fragments, not a compilable package. Their contents remain unchanged.
To reconstruct an original run, copy each archived file to its `original_path`
from the inventory, restoring the `.go` extension. Keeping these historical
inputs under `testdata` prevents them from becoming ordinary package tests.
They are source material for the audit, not a new ticket-validation gate.

The original scripts use fixed snapshot paths, manifests and hashes. They are
preserved without silently changing those identities. They require the original
inputs or an explicit adaptation of those paths to run elsewhere. For new work,
use the documented revision-comparison and compile-corpus commands linked above.
Do not automatically restart the historical run.

The #572 audit completed 656 of 934 designs before it was stopped. Its
classification records the accepted output differences and coverage boundary;
the interrupted shard is not counted as completed. This was partial evidence,
not a passing full-corpus run.

The current raw results, parent and final candidate source snapshots, and
remaining temporary source files were also preserved locally at
`/Users/luca/loom-verification-archive/572-20261001T032043Z`. Its `manifest.json`
records 81,175 regular files checked against their originals with SHA-256 and
two checked symlink targets. This local evidence copy is not required to run the
repository's reusable harnesses. Earlier cleanup removed some older generated
outputs; the preservation copy does not claim to recover those historical runs.

Full-corpus audits are optional work selected for major changes. Normal ticket
validation uses direct regressions, affected packages and representative
generated-output checks, as specified in [AGENTS.md](../../../AGENTS.md).
