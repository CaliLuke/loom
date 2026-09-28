# Integration subprocess ownership

`ProcessLease.tla` models one parent, a guardian, one target child, one
descendant, and a child launch that can complete after parent termination.
The guardian must exist before launch. Only the parent retains the lease writer
after exec. A forked, pending child temporarily holds a close-on-exec copy until
it has joined the group; this prevents EOF during group binding. Parent death
or cancellation closes the parent copy. EOF makes the guardian terminate its
process group, including itself. A launch either joins before termination or
fails without leaving a live child.

Run with Java and the TLA+ tools JAR (no repository dependency is added):

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg ProcessLease.tla
```

The configurations exercise the same state machine:

| Configuration | Result |
| --- | --- |
| `legacy.cfg` | Ownership invariant fails: parent dies with a live, unowned child. |
| `late-guard.cfg` | Ownership fails when the parent dies between child launch and guardian creation. |
| `inherited-lease.cfg` | Cleanup liveness fails when descendants retain the lease writer. |
| `checked.cfg` | Ownership and eventual cleanup pass: 48 generated, 23 distinct states. |

`Owned` requires a live parent or guardian for every live child. `Reaped`
requires eventual absence of children and pending launches after lease closure.
Weak fairness assumes that an enabled launch and guardian cleanup eventually
execute. The normal child-exit transition includes releasing the parent's
lease after `Wait`; the implementation bounds inherited-output waits with
`exec.Cmd.WaitDelay`.

Go 1.27 applies `Setpgid` before exec closes inherited descriptors in
`syscall/exec_bsd.go` and `syscall/exec_linux.go`. The model assumes this ordering
for a pending fork.

This is a finite protocol check, not a proof of Unix or Go. It abstracts group
termination as atomic, assumes the guardian is not independently killed and
descendants do not detach, and excludes PID reuse, fork bombs and OS failures.
Runtime tests cover actual descriptor inheritance, group joining, cancellation,
normal exit, retained output pipes, failed launches, SIGKILL of the parent,
and the JSON-RPC test-timeout panic that originally leaked its CLI.
