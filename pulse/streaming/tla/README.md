# TLA+ model of `Sink.AddStream` in `pulse/streaming`

`SinkAddStream.tla` models how instances of one sink add a stream to the sink
and remove it again (issue #481). Every in-process instance of a sink has its
own consumer. The replicated consumers map of a stream holds, under the sink
name, the consumers of the instances that added the stream:

- `Sink.AddStream` adds the consumer to the map and creates the consumer
  group. It keeps an existing group (`BUSYGROUP`).
- `Sink.RemoveStream` removes the consumer from the map and destroys the group
  when no consumer remains.

Each map update and each group command is one atomic Redis step. The steps of
one call are not atomic together, so the calls of two instances interleave.
Group creation can fail at any time.

## Invariants

| Invariant | Meaning |
| --- | --- |
| `NoStaleMember` | The map holds no consumer of an instance that has not added the stream. |
| `AddedHasGroup` | An instance whose `AddStream` succeeded reads from an existing group. |

## Toggles

| Constant | `TRUE` means |
| --- | --- |
| `CREATE_FIRST` | `AddStream` creates the group before it adds the consumer to the map. |
| `ROLLBACK` | `AddStream` removes the consumer from the map again when the group creation fails. |
| `ATOMIC_REMOVE` | `RemoveStream` removes the consumer and destroys an unused group in one step. |

## Running TLC

TLC ships in `tla2tools.jar` (tested with TLC 2.19).

```sh
curl -L -o /tmp/tla2tools.jar \
  https://github.com/tlaplus/tlaplus/releases/latest/download/tla2tools.jar
cd pulse/streaming/tla
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/tlc-states \
  -config cfg/fixed.cfg SinkAddStream.tla
```

`-deadlock` is required: a state with no enabled action is a normal end state.
`-metadir` keeps the `states/` directory out of the source tree. Every
configuration finishes in about a second.

## Results

| Configuration | Result |
| --- | --- |
| `cfg/asis.cfg` | `NoStaleMember` is violated after a 4-state trace: the instance adds its consumer to the map, the group creation fails, and the consumer stays in the map. This is issue #481. |
| `cfg/create_first.cfg` | `AddedHasGroup` is violated after an 8-state trace: instance 2 creates the group (`BUSYGROUP`), instance 1 removes the stream, sees no remaining consumer and destroys the group, then instance 2 adds its consumer and returns with no group. Moving the map update after the group creation is therefore not a fix. |
| `cfg/fixed_nonatomic.cfg` | The fix of #481 as shipped, with `RemoveStream` as on `main`. `NoStaleMember` holds with three instances (424 distinct states). The consumer goes to the map before the group creation, so a concurrent `RemoveStream` sees it and keeps the group, and a failed creation removes it again. `AddedHasGroup` is not checked here because of the `RemoveStream` race below. |
| `cfg/fixed_remove_asis.cfg` | The same design, checking `AddedHasGroup`: it is violated after a 9-state trace, with or without the fix of #481. `RemoveStream` removes the last consumer, another instance adds the stream and finds the existing group, then the first instance destroys the group. The map update and the group destroy of `RemoveStream` are separate calls (issue #508). |
| `cfg/fixed.cfg` | The fix of #481 with a future atomic `RemoveStream` (#508). Both invariants hold with three instances (189 distinct states). |

The fix does not destroy the group when its rollback leaves no consumer in the
map. A destroy there would reproduce the `RemoveStream` race above for
instances that add the stream at the same time. The group that such a rollback
can leave is the one an earlier instance created. The next `AddStream` reuses
it.
