package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
)

func sortRows(rows []table.Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a[colType] == b[colType] {
			return strings.ToLower(a[colName]) < strings.ToLower(b[colName])
		}
		return a[colType] == rowTypeDir
	})
}

func formatSize(entry entry) string {
	if entry.isDir {
		return ""
	}
	return humanSize(entry.size)
}

var sizeUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

func humanSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	value := float64(bytes)
	idx := 0
	for value >= 1024 && idx < len(sizeUnits)-1 {
		value /= 1024
		idx++
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f%s", value, sizeUnits[idx])
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", value), "0"), ".") + sizeUnits[idx]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
