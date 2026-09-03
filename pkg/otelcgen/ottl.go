// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"fmt"
)

// attrRef returns the OTTL getter expression for a resource attribute, e.g. attributes["foo"].
func attrRef(key string) string {
	return fmt.Sprintf("attributes[%q]", key)
}

// setIfAbsent returns a "set(...) where target == nil" statement: it only ever fills a gap in
// target, so it never overwrites a value a higher-precedence source already set there.
func setIfAbsent(target, valueExpr string) string {
	ref := attrRef(target)

	return fmt.Sprintf("set(%s, %s) where %s == nil", ref, valueExpr, ref)
}

// setLiteralIfAbsent is setIfAbsent for a literal string value.
func setLiteralIfAbsent(target, literal string) string {
	return setIfAbsent(target, fmt.Sprintf("%q", literal))
}

// setLiteralIfPresentAndAbsent returns a statement that sets target to a fixed literal, but only
// when presenceKey holds a string (the related fact is present) and target isn't already claimed
// by a higher-precedence source.
func setLiteralIfPresentAndAbsent(target, literal, presenceKey string) string {
	return fmt.Sprintf("set(%s, %q) where IsString(%s) and %s == nil",
		attrRef(target), literal, attrRef(presenceKey), attrRef(target))
}

// setValueIfPresentAndAbsent returns a statement that copies source's value into target, but only
// when source holds a string (the related fact is present) and target isn't already claimed by a
// higher-precedence source.
func setValueIfPresentAndAbsent(target, source string) string {
	sourceRef := attrRef(source)

	return fmt.Sprintf("set(%s, %s) where IsString(%s) and %s == nil",
		attrRef(target), sourceRef, sourceRef, attrRef(target))
}

// deleteKeys returns one delete_key(...) statement per resource attribute key.
func deleteKeys(keys ...string) []string {
	statements := make([]string, len(keys))
	for i, key := range keys {
		statements[i] = fmt.Sprintf("delete_key(attributes, %q)", key)
	}

	return statements
}
