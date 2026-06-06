// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package common

import (
	"fmt"
	"time"
)

// FormatHMSMillis renders a duration as hh:mm:ss.xxx with millisecond
// precision and zero-padded fields (issue #311). Negative durations
// clamp to zero so a clock-skew or out-of-order Now() can never make
// the operator-visible log line stutter.
func FormatHMSMillis(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int64(d / time.Hour)
	m := int64(d % time.Hour / time.Minute)
	s := int64(d % time.Minute / time.Second)
	ms := int64(d % time.Second / time.Millisecond)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
