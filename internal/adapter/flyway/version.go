package flyway

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	// Matches V[version]__[description].sql
	// Group 1 is the [version] part
	flywayFileRegex = regexp.MustCompile(`^V([a-zA-Z0-9\._\-]+)__.*\.sql$`)
)

// GetNextVersion identifies the next Flyway version given a directory.
func GetNextVersion(dir string, isTimestampFallback bool) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read flyway directory: %w", err)
	}

	var maxVersion string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := flywayFileRegex.FindStringSubmatch(file.Name())
		if len(matches) > 1 {
			ver := matches[1]
			// A simplistic approach: lexicographical or numerical max
			// For mixed formats, you might need semantic version sorting.
			// This string comparison works for zero-padded numeric and timestamp strings.
			if ver > maxVersion {
				maxVersion = ver
			}
		}
	}

	if maxVersion == "" {
		if isTimestampFallback {
			return time.Now().Format("20060102150405"), nil
		}
		return "1", nil
	}

	return IncrementVersion(maxVersion), nil
}

// IncrementVersion increments the numerical tail of a version string.
// If it's pure numbers and length > 12, assume timestamp, so return new timestamp.
// E.g. "1.2.3" -> "1.2.4"
// E.g. "003" -> "004"
// E.g. "20230101150000" -> "20230101150001" (actually we'd just return now for timestamps usually, but
// adhering to incrementing or new time if time has passed).
// We'll enforce a strict rule for timestamp: if length >= 12, return current timestamp (if > old).
func IncrementVersion(version string) string {
	// 1. Is it a timestamp? Try numerical && len >= 12
	isNumeric := true
	for _, c := range version {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}
	if isNumeric && len(version) >= 12 {
		nowStr := time.Now().Format("20060102150405")
		if nowStr > version {
			return nowStr
		}
		// If they run multiple in same second (unlikely timestamp fallback)
		// we just increment the number.
	}

	// 2. Find the last continuous sequence of digits
	lastNumIdx := -1
	for i := len(version) - 1; i >= 0; i-- {
		if version[i] >= '0' && version[i] <= '9' {
			lastNumIdx = i
			break
		}
	}

	if lastNumIdx == -1 {
		// No numbers found, just append .1? Or 1?
		return version + ".1"
	}

	startNumIdx := lastNumIdx
	for i := lastNumIdx - 1; i >= 0; i-- {
		if version[i] >= '0' && version[i] <= '9' {
			startNumIdx = i
		} else {
			break
		}
	}

	// prefix + incremented num + suffix
	prefix := version[:startNumIdx]
	numStr := version[startNumIdx : lastNumIdx+1]
	suffix := version[lastNumIdx+1:]

	num, _ := strconv.ParseUint(numStr, 10, 64)
	num++

	// Keep zero padding
	formatStr := fmt.Sprintf("%%0%dd", len(numStr))
	newNumStr := fmt.Sprintf(formatStr, num)

	return prefix + newNumStr + suffix
}
