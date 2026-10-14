package rmap

import (
	"bytes"
	"maps"
	"strconv"
	"strings"
	"testing"
)

// FuzzPackedMessageRoundTrip checks that set and del notifications encoded
// like the Lua struct.pack("ic0ic0ic0") scripts apply exactly to a replica.
func FuzzPackedMessageRoundTrip(f *testing.F) {
	f.Add("alpha", "one", uint64(1), []byte(nil))
	f.Add("", "", uint64(1), []byte{0})
	f.Add("k:ey", `["a","b,c"]`, uint64(1<<63), []byte("trailing"))
	f.Add(revField, "ignored", uint64(2), []byte(nil))
	f.Add("\x00\xff", "\xff\xff\xff\xff", uint64(18446744073709551615), []byte{0xff, 0xff, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, key, value string, rev uint64, rest []byte) {
		packed := append(packedStrings(key), rest...)
		gotKey, gotRest, err := unpackString(packed)
		if err != nil || gotKey != key || !bytes.Equal(gotRest, rest) {
			t.Errorf("unpackString(%x) = %q, %x, %v; want %q, %x", packed, gotKey, gotRest, err, key, rest)
		}
		revision := strconv.FormatUint(rev, 10)
		reserved := key == "" || strings.Contains(key, "=")

		sm := testMap()
		if rev > 0 {
			sm.rev = rev - 1
		}
		beforeRev := sm.rev
		change, applied, err := sm.applyMessageLocked("set", packedStrings(mapProtocol, "test", revision, key, value))
		switch {
		case reserved || rev == 0:
			if applied || len(sm.content) != 0 || sm.rev != beforeRev {
				t.Errorf("set %q rev %d applied %t with content %v rev %d", key, rev, applied, sm.content, sm.rev)
			}
		case err != nil:
			t.Errorf("set %q=%q rev %d: %v", key, value, rev, err)
		default:
			if !applied || change != (mapChange{kind: EventChange, rev: rev}) || sm.rev != rev {
				t.Errorf("set %q rev %d = %+v applied %t, replica rev %d", key, rev, change, applied, sm.rev)
			}
			if got, ok := sm.content[key]; !ok || got != value {
				t.Errorf("set %q stored %q, want %q", key, got, value)
			}
		}
		if reserved || rev == 0 {
			return
		}

		deleteRev := strconv.FormatUint(rev+1, 10)
		change, applied, err = sm.applyMessageLocked("del", packedStrings(mapProtocol, "test", deleteRev, key))
		switch {
		case rev == ^uint64(0):
			if applied {
				t.Errorf("wrapped del revision applied for %q", key)
			}
		case err != nil:
			t.Errorf("del %q rev %s: %v", key, deleteRev, err)
		default:
			if !applied || change != (mapChange{kind: EventDelete, rev: rev + 1}) {
				t.Errorf("del %q = %+v applied %t", key, change, applied)
			}
			if _, ok := sm.content[key]; ok {
				t.Errorf("del %q left content %v", key, sm.content)
			}
		}
	})
}

// FuzzApplyMessage checks that arbitrary pub/sub payloads never panic and
// leave the replica untouched when rejected.
func FuzzApplyMessage(f *testing.F) {
	for _, op := range []string{"set", "del", "reset", "destroy", "bogus", ""} {
		f.Add(op, packedStrings("alpha", "one", "7"))
		f.Add(op, packedStrings("alpha", "7"))
		f.Add(op, []byte("7"))
		f.Add(op, []byte{})
		f.Add(op, []byte{0xff, 0xff, 0xff, 0xff, 'a'})
		f.Add(op, []byte{0xff, 0xff, 0xff, 0x7f})
		f.Add(op, []byte{5, 0, 0, 0, 'a'})
		f.Add(op, append(packedStrings("alpha", "one"), 0xff, 0xff, 0xff, 0xff))
	}
	f.Fuzz(func(t *testing.T, op string, data []byte) {
		sm := testMap()
		sm.rev = 3
		sm.content["alpha"] = "zero"
		before := maps.Clone(sm.content)

		change, applied, err := sm.applyMessageLocked(op, data)

		if err != nil || !applied {
			if applied || change.kind != 0 || change.rev != 0 || sm.rev != 3 || !maps.Equal(before, sm.content) {
				t.Errorf("rejected %s %x mutated replica: change %+v applied %t err %v rev %d content %v", op, data, change, applied, err, sm.rev, sm.content)
			}
			return
		}
		if change.rev <= 3 || sm.rev != change.rev {
			t.Errorf("applied %s %x with change %+v and replica rev %d", op, data, change, sm.rev)
		}
	})
}
