package util

import (
	"strconv"
	"strings"
)

func CheckArr(input string) bool {
	input = strings.TrimSpace(input)
	lines := strings.Split(input, "\n")
	if len(lines) != ConfigInfo.Matrix.Width {
		return false
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		cols := strings.Split(line, " ")
		if len(cols) != ConfigInfo.Matrix.Length {
			return false
		}
		for _, col := range cols {
			_, err := strconv.Atoi(strings.TrimSpace(col))
			if err != nil {
				return false
			}
		}
	}
	return true
}
