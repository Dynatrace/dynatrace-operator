package maps

func ContainsKey[KeyType comparable, ValueType any](inputMap map[KeyType]ValueType, key KeyType) bool {
	for mapKey := range inputMap {
		if mapKey == key {
			return true
		}
	}

	return false
}
