// ABOUTME: orderedMap preserves insertion order for the --set resolved-fields
// ABOUTME: map, matching Python dict iteration used by the field: old -> new narration.
package status

// orderedMap is a string->string map that remembers insertion order. A key set
// more than once keeps its first position (matching Python dict assignment).
type orderedMap struct {
	order  []string
	values map[string]string
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: map[string]string{}}
}

func (m *orderedMap) set(key, val string) {
	if _, ok := m.values[key]; !ok {
		m.order = append(m.order, key)
	}
	m.values[key] = val
}

func (m *orderedMap) get(key string) (string, bool) {
	v, ok := m.values[key]
	return v, ok
}

func (m *orderedMap) keys() []string { return m.order }
