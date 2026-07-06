package rmap

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Set sets the value for the given key and returns the previous value.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// Set(ctx, "color", "blue") would set the "color" key to "blue"
// and return the previous value, if any.
func (sm *Map) Set(ctx context.Context, key, value string) (string, error) {
	res, err := sm.runSetScript(ctx, key, value)
	if err != nil {
		return "", err
	}
	return res.prev, nil
}

// SetEx sets the value for the given key and returns the previous value along
// with a flag indicating whether the key previously existed.
func (sm *Map) SetEx(ctx context.Context, key, value string) (string, bool, error) {
	res, err := sm.runSetScript(ctx, key, value)
	if err != nil {
		return "", false, err
	}
	return res.prev, res.existed, nil
}

// SetAndWait is a convenience method that calls Set and waits until the local
// replica has observed the committed revision. Multiple concurrent calls with
// the same key and value are allowed because each waiter is tied to its own
// committed write rather than the final key/value pair.
//
// The method will return an error if:
// - The key is empty
// - The key contains an equal sign
// - The context is cancelled
// - The map is stopped
// - There's an issue with the Redis operation
func (sm *Map) SetAndWait(ctx context.Context, key, value string) (string, error) {
	res, err := sm.runSetScript(ctx, key, value)
	if err != nil {
		return "", err
	}

	notifyCh := make(chan struct{}, 1)
	waitCtx, cancel := context.WithCancel(ctx)
	waiter := &setWaiter{
		ch:     notifyCh,
		rev:    res.rev,
		ctx:    waitCtx,
		cancel: cancel,
	}
	defer func() {
		cancel()
		sm.removeWaiter(waiter)
	}()
	if sm.registerWaiter(waiter) {
		return res.prev, nil
	}

	// Wait for notification or context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-sm.done:
		return "", fmt.Errorf("pulse map: %s is stopped", sm.Name)
	case <-notifyCh:
		return res.prev, nil
	}
}

// SetIfNotExists sets the value for key only if it doesn't exist.
// Returns true if the value was set, false if the key already existed.
func (sm *Map) SetIfNotExists(ctx context.Context, key, value string) (bool, error) {
	v, err := sm.runLuaScript(ctx, "setIfNotExists", sm.setIfNotExistsScript, key, value)
	if err != nil {
		return false, err
	}
	return v.(int64) == 1, nil
}

// TestAndSet sets the value for the given key if the current value matches the
// given test value. The previous value is returned.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// TestAndSet(ctx, "color", "red", "blue") would set "color" to "blue"
// only if its current value is "red", and return the previous value.
func (sm *Map) TestAndSet(ctx context.Context, key, test, value string) (string, error) {
	prev, err := sm.runLuaScript(ctx, "testAndSet", sm.testAndSetScript, key, test, value)
	if err != nil {
		return "", err
	}
	if prev == nil {
		return "", nil
	}
	return prev.(string), nil
}

// TestAndSetEx sets the value for the given key if the current value matches
// the given test value. It returns the previous value, whether the key existed
// and whether the value was updated.
func (sm *Map) TestAndSetEx(ctx context.Context, key, test, value string) (prev string, existed bool, updated bool, err error) {
	res, err := sm.runLuaScript(ctx, "testAndSet", sm.testAndSetScript, key, test, value)
	if err != nil {
		return "", false, false, err
	}
	if res == nil {
		return "", false, false, nil
	}
	prev = res.(string)
	existed = true
	updated = prev == test
	return prev, existed, updated, nil
}

// Inc increments the value for the given key and returns the result.
// The value must represent an integer.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - The value does not represent an integer
// - There's an issue with the Redis operation
//
// Example:
// Inc(ctx, "counter", 1) would increment the "counter" by 1
// and return the new value.
func (sm *Map) Inc(ctx context.Context, key string, delta int) (int, error) {
	res, err := sm.runLuaScript(ctx, "incr", sm.incrScript, key, delta)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(res.(string), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("pulse map: %s: %w", key, err)
	}
	return int(v), nil
}

// AppendValues appends the given items to the value for the given key and
// returns the result. The array of items is stored as a JSON array.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// AppendValues(ctx, "fruits", "apple", "banana") would append "apple" and "banana"
// to the existing list of fruits and return the updated list.
func (sm *Map) AppendValues(ctx context.Context, key string, items ...string) ([]string, error) {
	args := make([]any, 1+len(items))
	args[0] = key
	for i, item := range items {
		args[i+1] = item
	}
	res, err := sm.runLuaScript(ctx, "append", sm.appendScript, args...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return parseValues(res.(string)), nil
}

// AppendUniqueValues appends the given items to the value for the given key if
// they are not already present and returns the result. The array of items is
// stored as a JSON array.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// AppendUniqueValues(ctx, "fruits", "apple", "banana") would append only unique values
// to the existing list of fruits and return the updated list.
func (sm *Map) AppendUniqueValues(ctx context.Context, key string, items ...string) ([]string, error) {
	args := make([]any, 1+len(items))
	args[0] = key
	for i, item := range items {
		args[i+1] = item
	}
	res, err := sm.runLuaScript(ctx, "appendUnique", sm.appendUniqueScript, args...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return parseValues(res.(string)), nil
}

// RemoveValues removes the given items from the value for the given key and
// returns the remaining values after removal. The function behaves as follows:
//
//  1. The value for the key is expected to be a JSON array of items.
//  2. It removes all occurrences of the specified items from this list.
//  3. If the removal results in an empty list, the key is automatically deleted.
//  4. Returns the remaining items as a slice of strings, a boolean indicating
//     whether any value was removed, and an error (if any).
//  5. If the key doesn't exist, it returns nil, false, nil.
//
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// Given a key "fruits" with value ["apple","banana","cherry","apple"]
// RemoveValues(ctx, "fruits", "apple", "cherry") would return (["banana"], true, nil)
// and update the value in Redis to ["banana"]
func (sm *Map) RemoveValues(ctx context.Context, key string, items ...string) ([]string, bool, error) {
	args := make([]any, 1+len(items))
	args[0] = key
	for i, item := range items {
		args[i+1] = item
	}
	res, err := sm.runLuaScript(ctx, "remove", sm.removeScript, args...)
	if err != nil {
		return nil, false, err
	}
	result := res.([]any)
	if result[0] == nil {
		return nil, false, nil // Key didn't exist
	}
	remaining := result[0].(string)
	removed := result[1] != nil && result[1].(int64) == 1
	if removed && remaining == "" {
		return nil, true, nil // All items were removed, key was deleted
	}
	return parseValues(remaining), removed, nil
}

// Delete deletes the value for the given key and returns the previous value.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// Delete(ctx, "color") would delete the "color" key and return its previous value, if any.
func (sm *Map) Delete(ctx context.Context, key string) (string, error) {
	prev, err := sm.runLuaScript(ctx, "delete", sm.delScript, key)
	if err != nil {
		return "", err
	}
	if prev == nil {
		return "", nil
	}
	return prev.(string), nil
}

// DeleteEx deletes the value for the given key and returns the previous value
// along with a flag indicating whether the key existed.
func (sm *Map) DeleteEx(ctx context.Context, key string) (string, bool, error) {
	prev, err := sm.runLuaScript(ctx, "delete", sm.delScript, key)
	if err != nil {
		return "", false, err
	}
	if prev == nil {
		return "", false, nil
	}
	return prev.(string), true, nil
}

// TestAndDelete tests that the value for the given key matches the test value
// and deletes the key if it does. It returns the previous value.
// An error is returned if:
// - The key is empty
// - The key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// TestAndDelete(ctx, "color", "blue") would delete the "color" key only if
// its current value is "blue", and return the previous value.
func (sm *Map) TestAndDelete(ctx context.Context, key, test string) (string, error) {
	prev, err := sm.runLuaScript(ctx, "testAndDelete", sm.testAndDelScript, key, test)
	if err != nil {
		return "", err
	}
	if prev == nil {
		return "", nil
	}
	return prev.(string), nil
}

// TestAndDeleteEx deletes the given key if its current value matches the given
// test value. It returns the previous value, whether the key existed and
// whether the key was deleted.
func (sm *Map) TestAndDeleteEx(ctx context.Context, key, test string) (prev string, existed bool, deleted bool, err error) {
	res, err := sm.runLuaScript(ctx, "testAndDelete", sm.testAndDelScript, key, test)
	if err != nil {
		return "", false, false, err
	}
	if res == nil {
		return "", false, false, nil
	}
	prev = res.(string)
	existed = true
	deleted = prev == test
	return prev, existed, deleted, nil
}

// Reset clears the map content. Reset remains available after Close because it
// mutates Redis state even though the local replica is already frozen.
func (sm *Map) Reset(ctx context.Context) error {
	_, err := sm.runLuaScript(ctx, "reset", sm.resetScript, "*")
	return err
}

// Destroy clears the map content from Redis and notifies subscribers while
// preserving the internal revision ordering used by live replicas.
func (sm *Map) Destroy(ctx context.Context) error {
	_, err := sm.runLuaScript(ctx, "destroy", sm.destroyScript, "*")
	return err
}

// TestAndReset tests that the values for the given keys match the test values
// and clears the map if they do. It returns true if the map was cleared, false otherwise.
// An error is returned if:
// - Any key is empty
// - Any key contains an equal sign
// - There's an issue with the Redis operation
//
// Example:
// TestAndReset(ctx, []string{"color", "size"}, []string{"blue", "large"}) would clear the map
// only if the "color" key has value "blue" and the "size" key has value "large",
// and return true if the map was cleared.
func (sm *Map) TestAndReset(ctx context.Context, keys, tests []string) (bool, error) {
	if len(keys) != len(tests) {
		return false, fmt.Errorf("pulse map: %s TestAndReset requires len(keys) == len(tests)", sm.Name)
	}
	for _, k := range keys {
		if len(k) == 0 {
			return false, fmt.Errorf("pulse map: %s key cannot be empty in %q", sm.Name, "testAndReset")
		}
		if strings.Contains(k, "=") {
			return false, fmt.Errorf("pulse map: %s key %q cannot contain '=' in %q", sm.Name, k, "testAndReset")
		}
	}

	args := make([]any, 1+len(keys)+len(tests))
	args[0] = "*"
	for i, k := range keys {
		args[i+1] = k
	}
	for i, t := range tests {
		args[len(keys)+i+1] = t
	}
	res, err := sm.runLuaScript(ctx, "testAndReset", sm.testAndResetScript, args...)
	if err != nil {
		return false, err
	}
	return res.(int64) == 1, nil
}
