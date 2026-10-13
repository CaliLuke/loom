# Subscriber admission after Close — #683

Parent: `6486dfb45`. Scope: Reader/Sink subscriber registration and Reader startup
at the existing lifecycle lock. No Redis dependency or #800 operation changes.

1. [x] Model legacy registration after shutdown and verify the counterexample;
   check a shared admission rule before production edits.
2. [x] Add failing tests for closed/closing Reader and Sink, Reader Close before
   first Subscribe, and no registration/worker publication after shutdown.
3. [x] Use one shared package-private subscriber registration helper under each
   owner's lock. Return an already-closed channel when closing/closed; only an
   admitted Reader subscriber can start its worker.
4. [x] Verify normal delivery/unsubscribe and concurrent Subscribe/Close on
   miniredis and Redis8.10.2, with focused race/package gates and model alignment.
5. [x] Freeze exact diff and evidence for fresh review. Root owns commit and
   integration; this prerequisite must land separately before #800 continues.

Acceptance: every Subscribe after shutdown admission is sealed returns an
already-closed channel without changing owner subscribers or starting a worker.
Subscriptions admitted first retain current delivery/shutdown behavior. This
corrects documented behavior; it introduces no new Close I/O deadline or join
contract. #800 remains separately blocked on upstream dependency fetchability.

Validation and model correspondence are recorded in
[pulse/streaming/tla/subscriber_admission](../pulse/streaming/tla/subscriber_admission/README.md).
Implementation restores the existing public contract and updates Subscribe GoDoc;
no shared consumer guide or master tracker was edited in this isolated task.

## Integration review

Independent review approved the original nine-file diff against `6486dfb45`
(SHA `637ba22f3ef882ffc485ce07aea63581b82db113d5fb51ed565e0cbf0cae2818`)
with no findings. The reviewer independently reran the admission race tests
(1.530s) and the passing 34-state model. Root integrated those exact source
files against `7b29289ae` and added public/contributor guidance.

The integrated subscriber admission race selection passes; its log is
`/tmp/loom-683-integrated.log`. Original full-package, real-Redis, concurrency
repeat and model results remain valid because production and test files are
byte-identical to the approved source. The final integration diff requires
review before commit.

Delivered as `ebc06ec94`, independently reviewed, pushed to main and issue closed.
