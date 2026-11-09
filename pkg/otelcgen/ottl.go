// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"fmt"
)

// DeleteKeys emits OTTL that delete the provided attribute keys.
func DeleteKeys(keys ...string) []string {
	statements := make([]string, len(keys))
	for i, key := range keys {
		statements[i] = deleteAttr(key)
	}

	return statements
}

// RenameAttr emits OTTL that copies the attribute value from source to target and deletes source.
func RenameAttr(target, source string) []string {
	return []string{
		fmt.Sprintf(`set(%[1]s, %[2]s) where %[2]s != nil`, attrRef(target), attrRef(source)),
		deleteAttr(source),
	}
}

// SetAttrFromEnv emits OTTL that copies the value from the env to the attribute key
// if the env is defined and the key is not set.
func SetAttrFromEnv(attrKey, envName string) string {
	return fmt.Sprintf(`set(%[1]s, %[2]q) where %[1]s == nil and Len(%[2]q) > 0`,
		attrRef(attrKey), fmt.Sprintf("${env:%s}", envName))
}

// SetAttrIfAbsent emits OTTL that copies the value from one attribute to another if the target is not defined.
func SetAttrIfAbsent(target, source string) string {
	return fmt.Sprintf("set(%[1]s, %[2]s) where %[1]s == nil", attrRef(target), attrRef(source))
}

func attrRef(key string) string {
	return fmt.Sprintf("attributes[%q]", key)
}

func setLiteralIfPresentAndAbsent(target, literal, presenceKey string) string {
	return fmt.Sprintf("set(%[1]s, %[2]q) where IsString(%[3]s) and %[1]s == nil",
		attrRef(target), literal, attrRef(presenceKey))
}

func setValueIfPresentAndAbsent(target, source string) string {
	return fmt.Sprintf("set(%[1]s, %[2]s) where IsString(%[2]s) and %[1]s == nil",
		attrRef(target), attrRef(source))
}

func deleteAttr(key string) string {
	return fmt.Sprintf("delete_key(attributes, %q)", key)
}
