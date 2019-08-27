package helm

import (
	"crypto/sha256"
	"fmt"

	"github.com/emosbaugh/yaml"
	"github.com/pkg/errors"
)

// MergeHelmValues merges user edited values from state file and vendor values from upstream Helm repo.
// base is the original config from state
// user is the modified config from state
// vendor is the new config from current chart
// Value priorities: user, vendor, base
func MergeHelmValues(baseValues, userValues, vendorValues string, preserveComments bool) (string, error) {
	// First time merge is performed, there are no user values.  We are shortcutting this
	// in order to preserve original file formatting and comments
	if userValues == "" {
		return vendorValues, nil
	}
	if vendorValues == "" {
		vendorValues = baseValues
	}

	var base, user, vendor yaml.MapSlice

	// we can drop comments in base
	if err := yaml.Unmarshal([]byte(baseValues), &base); err != nil {
		return "", errors.Wrapf(err, "unmarshal base values")
	}
	// TODO: preserve user comments
	if err := yaml.Unmarshal([]byte(userValues), &user); err != nil {
		return "", errors.Wrapf(err, "unmarshal user values")
	}
	if preserveComments {
		var unmarshaler yaml.CommentUnmarshaler
		if err := unmarshaler.Unmarshal([]byte(vendorValues), &vendor); err != nil {
			return "", errors.Wrapf(err, "unmarshal vendor values")
		}
	} else {
		if err := yaml.Unmarshal([]byte(vendorValues), &vendor); err != nil {
			return "", errors.Wrapf(err, "unmarshal vendor values")
		}
	}

	merged, err := deepMerge(base, user, vendor)
	if err != nil {
		return "", errors.Wrap(err, "deep merge values")
	}

	vals, err := yaml.Marshal(merged)
	if err != nil {
		return "", errors.Wrapf(err, "marshal merged values")
	}
	return string(vals), nil
}

// Value priorities: user, vendor, base
func deepMerge(base, user, vendor yaml.MapSlice) (yaml.MapSlice, error) {
	merged := yaml.MapSlice{}

	allKeys := getAllKeys(vendor, user) // we can drop keys that have been dropped by the vendor

	for _, k := range allKeys {
		// don't merge comments
		if _, ok := k.(yaml.Comment); ok {
			merged = append(merged, yaml.MapItem{Key: k})
			continue
		}

		baseVal, baseOk := getValueFromKey(base, k)
		userVal, userOk := getValueFromKey(user, k)
		vendorVal, vendorOk := getValueFromKey(vendor, k)

		numExistingMaps := 0
		preprocessValue := func(exists bool, value interface{}) yaml.MapSlice {
			if !exists {
				return yaml.MapSlice{}
			}
			m, ok := value.(yaml.MapSlice)
			if ok {
				numExistingMaps++
			}
			return m
		}

		baseSubmap := preprocessValue(baseOk, baseVal)
		userSubmap := preprocessValue(userOk, userVal)
		vendorSubmap := preprocessValue(vendorOk, vendorVal)

		if numExistingMaps > 1 {
			mergedSubmap, err := deepMerge(baseSubmap, userSubmap, vendorSubmap)
			if err != nil {
				return merged, errors.Wrapf(err, "merge submap at key %s", k)
			}
			merged = setValueAtKey(merged, k, mergedSubmap)
			continue
		}

		if userOk && baseOk && vendorOk {
			if eq, err := valuesEqual(userVal, baseVal); err != nil {
				return merged, errors.Wrapf(err, "compare values at key %s", k)
			} else if eq {
				// user didn't change the value shipped in the previous version
				// so we continue propagating vendor shipped values
				merged = setValueAtKey(merged, k, vendorVal)
			} else {
				merged = setValueAtKey(merged, k, userVal)
			}
		} else if userOk {
			merged = setValueAtKey(merged, k, userVal)
		} else {
			merged = setValueAtKey(merged, k, vendorVal)
		}
	}
	return merged, nil
}

func getAllKeys(maps ...yaml.MapSlice) (allKeys []interface{}) {
	seenKeys := map[interface{}]bool{}
	for _, m := range maps {
		for _, item := range m {
			// comments are unique
			if _, ok := item.Key.(yaml.Comment); ok {
				allKeys = append(allKeys, item.Key)
			} else if _, ok := seenKeys[item.Key]; !ok {
				seenKeys[item.Key] = true
				allKeys = append(allKeys, item.Key)
			}
		}
	}
	return
}

func getValueFromKey(m yaml.MapSlice, key interface{}) (interface{}, bool) {
	for _, item := range m {
		if item.Key == key {
			return item.Value, true
		}
	}
	return nil, false
}

func setValueAtKey(m yaml.MapSlice, key, value interface{}) (next yaml.MapSlice) {
	var found bool
	for _, item := range m {
		if item.Key == key {
			item.Value = value
			found = true
		}
		next = append(next, item)
	}
	if !found {
		next = append(next, yaml.MapItem{Key: key, Value: value})
	}
	return
}

func valuesEqual(val1, val2 interface{}) (bool, error) {
	val1Array, val1OK := val1.([]interface{})
	val2Array, val2OK := val2.([]interface{})
	if !val1OK && !val2OK {
		// arrays are the only types expected here that won't compare with `==`
		return val1 == val2, nil
	}
	if !val1OK || !val2OK {
		return false, nil
	}

	// compare two arrays by checksumming their elements
	arr1Checksums, err := arrayToChecksums(val1Array)
	if err != nil {
		return false, errors.Wrap(err, "array1 to checksums")
	}
	arr2Checksums, err := arrayToChecksums(val2Array)
	if err != nil {
		return false, errors.Wrap(err, "array2 to checksums")
	}

	for k := range arr1Checksums {
		_, ok := arr2Checksums[k]
		if !ok {
			return false, nil
		}
		delete(arr2Checksums, k)
	}

	return len(arr2Checksums) == 0, nil
}

func arrayToChecksums(arr []interface{}) (map[string]bool, error) {
	result := map[string]bool{}
	for _, v := range arr {
		str, err := yaml.Marshal(v)
		if err != nil {
			return nil, errors.Wrap(err, "marshal value into yaml")
		}
		sum := sha256.Sum256(str)
		result[fmt.Sprintf("%x", sum)] = true
	}
	return result, nil
}
