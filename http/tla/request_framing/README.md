# Request framing ownership model

Pinned TLC1.7.4 (TLC2.19), reused installed jar
`/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`.
Run `java -cp <jar> tlc2.TLC -config owned.cfg RequestFraming.tla`.

This finite model concerns an explicitly declared zero length with an opaque
reader. Current publication never establishes an empty body; the current mode
reproduces that mismatch. Two rejected repairs independently expose double Close
when cancellation and the probe own cleanup separately, and a lost Close failure.
The owned variant shares one captured-body close decision, waits for completion,
retains read/close/context causes and publishes only proven EOF.

Implementation correspondence: Start/Read is a synchronous one-byte EOF probe;
ClaimClose is sync.Once over the original captured Body, shared by probe cleanup
and context.AfterFunc; FinishClose completion is observed before reading its
error or returning. Finalize stops/detaches cancellation and combines errors;
Publish represents successful preparation followed by the existing
context-bearing net/http publication boundary. A cancel observed after prepared
but before publication is the existing transport context gate, not a new custom
atomic network operation. Cancellation after publication belongs to net/http and
is outside this model.

Assumptions: Request.Body obeys Go1.27.2's documented concurrent Read/Close and
Close-unblocks-Read contract; Close terminates (its error is preserved), callbacks
and runnable transitions eventually execute. An empty/data/error reader resolves
its read; a blocked reader resolves after Close. The model does not prove wall
clock bounds for hostile Readers/Closers that violate that contract, codec or
header grammar, positive streaming length checks, retry bytes, arbitrary custom
Doer behavior, or actual syscall scheduling. A nonempty probe produces failure;
it is never replaced by a successful empty request. The model records only
whether a byte exists, not byte content.

Tests must check correspondence: Close once on EOF/read failure/nonempty/context
races, cancellation unblocks a coordinated reader with no detached probe worker,
all error causes survive, GetBody becomes an empty replay only after proven EOF,
and failed preparation never calls Do. Failed ordinary encoding also releases the
attached request body; that sequential cleanup is checked outside this zero-only
model. Positive streams stay streaming and net/http checks their declared length.

The runtime correspondence is `http/request_encoding.go`: `requestPreparation`
owns the captured body and `closeOnce`; `verifyEmpty` waits for cancellation
callback completion; `probeEmpty` treats only literal `io.EOF` as successful EOF.
Repeated `(0, nil)` reads stop with `io.ErrNoProgress` after 100 attempts, a
sequential progress bound outside the finite model. `request_encoding_test.go`
checks these seams, including joined EOF/error causes and callback completion.
The generated HTTP/JSON-RPC request-content-length tests cover publication after
body codecs, physical ContentLength, GetBody, and no Do on preparation failure.

Readiness results: current PublicationSafe counterexample (27 states), rejected
separate closers CloseOnce (124), rejected lost-close FailuresRetained (114),
owned 206 generated/137 distinct states with invariants and cancellation release.
Sources/configurations are unchanged from the approved readiness run; logs and
checker states are temporary evidence, not committed model inputs.

Method representation is admitted before this zero-probe lifecycle: only
POST/PUT/PATCH preserve explicit zero in pinned Go HTTP/1 and HTTP/2. Other
methods fail before Read/Do, using the same once-only failure cleanup. This
sequential admission is covered by direct and actual generated HTTP/1+HTTP/2
method tests, not by extra states in this concurrency model.
