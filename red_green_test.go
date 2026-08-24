package main

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/pkg/process"
	"github.com/codesandbox/codesandbox/pkg/stringutil"
)

func TestRedGreen(t *testing.T) {
	var failures []string

	chineseText := "你好世界测试中文编码"
	textBytes := []byte(chineseText)

	for i := 1; i < len(chineseText); i++ {
		r, size := utf8.DecodeRune(textBytes[i:])
		if r == utf8.RuneError && size == 1 {
			byteVal := textBytes[i]
			if byteVal >= 0x80 && byteVal <= 0xBF {
				continue
			}
		}
		break
	}

	rb := process.NewRingBuffer(1024)
	rb.Write(textBytes)
	got := rb.String()
	if got != chineseText {
		failures = append(failures, fmt.Sprintf("RingBuffer.Write/String corrupted: got %q (len %d), want %q (len %d)", got, len(got), chineseText, len(chineseText)))
	}

	rb2 := process.NewRingBuffer(1024)
	rb2.WriteAt(textBytes, 0)
	got2 := rb2.String()
	if got2 != chineseText {
		failures = append(failures, fmt.Sprintf("RingBuffer.WriteAt corrupted: got %q (len %d), want %q (len %d)", got2, len(got2), chineseText, len(chineseText)))
	}

	rb3 := process.NewRingBuffer(1024)
	rb3.WriteBytesSegment(textBytes, 0)
	got3 := rb3.String()
	if got3 != chineseText {
		failures = append(failures, fmt.Sprintf("RingBuffer.WriteBytesSegment corrupted: got %q (len %d), want %q (len %d)", got3, len(got3), chineseText, len(chineseText)))
	}

	sliced := rb3.SliceByteRange(0, len(chineseText))
	if sliced != chineseText {
		failures = append(failures, fmt.Sprintf("SliceByteRange corrupted: got %q (len %d), want %q (len %d)", sliced, len(sliced), chineseText, len(chineseText)))
	}

	truncated := stringutil.Truncate(chineseText, 5)
	expectedTrunc := "你好..."
	if truncated != expectedTrunc {
		failures = append(failures, fmt.Sprintf("stringutil.Truncate at 5 corrupted: got %q (len %d), want %q (len %d)", truncated, len(truncated), expectedTrunc, len(expectedTrunc)))
	}

	truncated3 := stringutil.Truncate(chineseText, 3)
	expectedTrunc3 := "你好世"
	if truncated3 != expectedTrunc3 {
		failures = append(failures, fmt.Sprintf("stringutil.Truncate at 3 corrupted: got %q (len %d), want %q (len %d)", truncated3, len(truncated3), expectedTrunc3, len(expectedTrunc3)))
	}

	byteTruncated := stringutil.ByteTruncate(chineseText, 7)
	expectedByteTrunc := "你好世"
	if byteTruncated != expectedByteTrunc {
		failures = append(failures, fmt.Sprintf("stringutil.ByteTruncate at 7 corrupted: got %q (len %d), want %q (len %d)", byteTruncated, len(byteTruncated), expectedByteTrunc, len(expectedByteTrunc)))
	}

	byteTruncated2 := stringutil.ByteTruncate(chineseText, 10)
	expectedByteTrunc2 := "你好世界"
	if byteTruncated2 != expectedByteTrunc2 {
		failures = append(failures, fmt.Sprintf("stringutil.ByteTruncate at 10 corrupted: got %q (len %d), want %q (len %d)", byteTruncated2, len(byteTruncated2), expectedByteTrunc2, len(expectedByteTrunc2)))
	}

	substring := stringutil.Substring(chineseText, 0, 7)
	expectedSub := "你好世"
	if substring != expectedSub {
		failures = append(failures, fmt.Sprintf("stringutil.Substring(0,7) corrupted: got %q (len %d), want %q (len %d)", substring, len(substring), expectedSub, len(expectedSub)))
	}

	substring2 := stringutil.Substring(chineseText, 3, 10)
	expectedSub2 := "好世"
	if substring2 != expectedSub2 {
		failures = append(failures, fmt.Sprintf("stringutil.Substring(3,10) corrupted: got %q (len %d), want %q (len %d)", substring2, len(substring2), expectedSub2, len(expectedSub2)))
	}

	sanitized := service.SanitizeOutput(chineseText, 30)
	if sanitized != chineseText {
		failures = append(failures, fmt.Sprintf("SanitizeOutput full corrupted: got %q (len %d), want %q (len %d)", sanitized, len(sanitized), chineseText, len(chineseText)))
	}

	sanitizedTrunc := service.SanitizeOutput(chineseText, 5)
	expectedSanitized := "你好..."
	if sanitizedTrunc != expectedSanitized {
		failures = append(failures, fmt.Sprintf("SanitizeOutput truncated at 5 corrupted: got %q (len %d), want %q (len %d)", sanitizedTrunc, len(sanitizedTrunc), expectedSanitized, len(expectedSanitized)))
	}

	if len(failures) > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		for _, f := range failures {
			fmt.Printf("  - %s\n", f)
		}
		t.FailNow()
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}