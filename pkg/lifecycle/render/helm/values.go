package helm

import (
	"crypto/sha256"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// Merges user edited values from state file and vendor values from upstream Helm repo.
// base is the originl config from state
// user is the modified config from state
// vendor is the new config from current chart
// Value priotities: user, vendor, base
func MergeHelmValues(baseValues, userValues, vendorValues string) (string, error) {
	base := map[string]interface{}{}
	user := map[string]interface{}{}
	vendor := map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(baseValues), &base); err != nil {
		return "", errors.Wrapf(err, "unmarshal base values")
	}
	if err := yaml.Unmarshal([]byte(userValues), &user); err != nil {
		return "", errors.Wrapf(err, "unmarshal user values")
	}
	if err := yaml.Unmarshal([]byte(vendorValues), &vendor); err != nil {
		return "", errors.Wrapf(err, "unmarshal vendor values")
	}

	merged := map[string]interface{}{}
	deepMerge(base, user, vendor, merged)

	vals, err := yaml.Marshal(merged)
	if err != nil {
		return "", errors.Wrapf(err, "marshal merged values")
	}
	return string(vals), nil
}

// Value priotities: user, vendor, base
func deepMerge(base, user, vendor, merged map[string]interface{}) error {
	allKeys := getAllKeys(base, user, vendor)
	for _, k := range allKeys {
		baseVal, baseOk := base[k]
		userVal, userOk := user[k]
		vendorVal, vendorOk := vendor[k]

		numExistingMaps := 0
		preprocessValue := func(exists bool, value interface{}) map[string]interface{} {
			if !exists {
				return map[string]interface{}{}
			}
			if m, ok := value.(map[interface{}]interface{}); ok {
				numExistingMaps += 1
				return makeStringMap(m)
			}
			return map[string]interface{}{}
		}

		baseSubmap := preprocessValue(baseOk, baseVal)
		userSubmap := preprocessValue(userOk, userVal)
		vendorSubmap := preprocessValue(vendorOk, vendorVal)

		if numExistingMaps > 1 {
			mergedSubmap := map[string]interface{}{}
			deepMerge(baseSubmap, userSubmap, vendorSubmap, mergedSubmap)
			merged[k] = mergedSubmap
			continue
		}

		if userOk && baseOk && vendorOk {

			if eq, err := valuesEqual(userVal, baseVal); err != nil {
				return errors.Wrapf(err, "compare values at key %s", k)
			} else if eq {
				// user didn't change the value shipped in the previous version
				// so we continue propagating vendor shipped values
				merged[k] = vendorVal
			} else {
				merged[k] = userVal
			}
		} else if userOk {
			merged[k] = userVal
		} else if vendorOk {
			merged[k] = vendorVal
		} else {
			merged[k] = baseVal // vendor stopped shipping this value?
		}
	}
	return nil
}

func getAllKeys(maps ...map[string]interface{}) []string {
	allKeys := map[string]bool{}
	for _, m := range maps {
		for k, _ := range m {
			allKeys[k] = true
		}
	}

	keys := make([]string, len(allKeys), len(allKeys))
	i := 0
	for k, _ := range allKeys {
		keys[i] = k
		i += 1
	}
	return keys
}

func makeStringMap(m map[interface{}]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		result[k.(string)] = v
	}
	return result
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

	for k, _ := range arr1Checksums {
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
