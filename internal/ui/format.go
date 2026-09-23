package ui

import (
	"fmt"
)

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatNumber(n int64) string {
	in := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(in)+len(in)/3)
	leading := len(in) % 3
	if leading > 0 {
		out = append(out, in[:leading]...)
		if len(in) > 3 {
			out = append(out, ',')
		}
	}
	for i := leading; i < len(in); i += 3 {
		out = append(out, in[i:i+3]...)
		if i+3 < len(in) {
			out = append(out, ',')
		}
	}
	return string(out)
}
