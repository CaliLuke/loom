package ir

import "slices"

// acquisitionJournal owns index changes made by one outer schema scope. Nested
// analysis and held completion append to the same journal. Prior published
// indexes stay in place; checkpoint cost is independent of their cardinality.
type acquisitionJournal struct {
	undo []func()
}

// acquisitionRollback begins a scope and returns its reverse-order rollback.
// Map entries and lazily allocated indexes are restored; user callbacks and
// mutable schema bodies retain the existing acquisition contract.
func (a *Analyzer) acquisitionRollback() func() {
	journal := &acquisitionJournal{}
	a.acquisitionJournal = journal
	return func() {
		a.acquisitionJournal = nil
		for index := len(journal.undo) - 1; index >= 0; index-- {
			journal.undo[index]()
		}
		journal.undo = nil
	}
}

// setAcquisitionEntry records the immediately preceding value before writing.
// Repeated writes unwind in reverse order. Mutable values must be detached
// before replacement. appendAcquisitionValues also protects spare capacity.
func setAcquisitionEntry[K comparable, V any](a *Analyzer, index map[K]V, key K, value V) {
	if journal := a.acquisitionJournal; journal != nil {
		prior, present := index[key]
		journal.undo = append(journal.undo, func() {
			if present {
				index[key] = prior
			} else {
				delete(index, key)
			}
		})
	}
	index[key] = value
}

// appendAcquisitionValues detaches an active scope's append from prior capacity:
// an appended binding must not remain reachable beyond the restored slice length
// after rollback. Arguments retain the original prefix-before-callback order.
func appendAcquisitionValues[V any](a *Analyzer, values []V, value V) []V {
	if a.acquisitionJournal != nil {
		values = slices.Clip(values)
	}
	return append(values, value)
}

func deleteAcquisitionEntry[K comparable, V any](a *Analyzer, index map[K]V, key K) {
	prior, present := index[key]
	if !present {
		return
	}
	if journal := a.acquisitionJournal; journal != nil {
		journal.undo = append(journal.undo, func() {
			index[key] = prior
		})
	}
	delete(index, key)
}

// ensureAcquisitionIndex preserves nil versus allocated state on failure.
func ensureAcquisitionIndex[K comparable, V any](a *Analyzer, index *map[K]V) {
	if *index != nil {
		return
	}
	if journal := a.acquisitionJournal; journal != nil {
		journal.undo = append(journal.undo, func() {
			*index = nil
		})
	}
	*index = make(map[K]V)
}
