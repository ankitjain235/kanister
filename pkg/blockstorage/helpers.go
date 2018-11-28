package blockstorage

import (
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

// SanitizeTags are used to sanitize the tags
func SanitizeTags(tags map[string]string) map[string]string {
	// From https://cloud.google.com/compute/docs/labeling-resources
	// - Keys and values cannot be longer than 63 characters each.
	// - Keys and values can only contain lowercase letters, numeric
	//   characters, underscores, and dashes. International characters
	//   are allowed.
	// - Label keys must start with a lowercase letter and international
	//   characters are allowed.
	fixedTags := make(map[string]string)
	for k, v := range tags {
		fixedTags[ktags.SanitizeValueForGCP(k)] = ktags.SanitizeValueForGCP(v)
	}
	return fixedTags
}

// KeyValueToMap converts a KeyValue slice to a map
func KeyValueToMap(kv []*KeyValue) map[string]string {
	stringMap := make(map[string]string)
	for _, t := range kv {
		stringMap[t.Key] = t.Value
	}
	return stringMap
}
