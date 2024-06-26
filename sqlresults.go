package main

import (
	"slices"
	"strings"
)

func ParseSQLResults(out string) []string {
	results := strings.Split(out, "\n")
	// headerの後に出力される --- を削除
	if len(results) >= 2 && strings.HasPrefix(results[1], "-") {
		results = slices.Delete(results, 1, 2)
	}
	// 末尾の空行を削除
	if results[len(results)-1] == "" {
		results = slices.Delete(results, len(results)-1, len(results))
	}
	return results
}
