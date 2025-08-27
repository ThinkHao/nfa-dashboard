package controller

import "strconv"

// parseIntDefault 解析正整数，非法或空返回默认值
func parseIntDefault(s string, def int) int {
    if s == "" { return def }
    if v, err := strconv.Atoi(s); err == nil && v > 0 { return v }
    return def
}
