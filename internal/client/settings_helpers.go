package client

import "strings"

// buildNestedMap converts a flat map with dotted keys into a nested map.
// e.g. {"index.number_of_replicas": "2"} becomes {"index": {"number_of_replicas": "2"}}
// Multiple keys sharing a common prefix are merged rather than overwritten.
func buildNestedMap(flat map[string]any) map[string]any {
	result := make(map[string]any)
	for key, val := range flat {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) == 1 {
			result[key] = val
			continue
		}
		sub, ok := result[parts[0]].(map[string]any)
		if !ok {
			sub = make(map[string]any)
			result[parts[0]] = sub
		}
		for k, v := range buildNestedMap(map[string]any{parts[1]: val}) {
			if existingSub, ok := sub[k].(map[string]any); ok {
				if newSub, ok := v.(map[string]any); ok {
					mergeNestedMaps(existingSub, newSub)
					continue
				}
			}
			sub[k] = v
		}
	}
	return result
}

// mergeNestedMaps merges src into dst recursively. When both dst[k] and src[k]
// are maps, they are merged; otherwise src[k] overwrites dst[k].
func mergeNestedMaps(dst, src map[string]any) {
	for k, v := range src {
		if existingSub, ok := dst[k].(map[string]any); ok {
			if newSub, ok := v.(map[string]any); ok {
				mergeNestedMaps(existingSub, newSub)
				continue
			}
		}
		dst[k] = v
	}
}
