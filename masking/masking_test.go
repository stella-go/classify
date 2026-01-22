// Copyright 2010-NOW the original author or authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package masking

import (
	"fmt"
	"math"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple empty", "", ModeSimple, ""},
		{"simple hello", "hello", ModeSimple, "*****"},
		{"pattern hello", "hello", ModePattern, "h***o"},
		{"pattern short", "ab", ModePattern, "**"},
		{"pattern single", "a", ModePattern, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("String(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestStringKeep(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		keepStart int
		keepEnd   int
		expected  string
	}{
		{"phone", "13812345678", 3, 4, "138****5678"},
		{"short", "abc", 1, 1, "a*c"},
		{"all keep", "abc", 2, 2, "abc"},
		{"empty", "", 3, 4, ""},
		{"negative start", "hello", -1, 1, "****o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringKeep(tt.input, tt.keepStart, tt.keepEnd)
			if result != tt.expected {
				t.Errorf("StringKeep(%q, %d, %d) = %q, want %q", tt.input, tt.keepStart, tt.keepEnd, result, tt.expected)
			}
		})
	}
}

func TestStringMiddle(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maskPercent float64
		expected    string
	}{
		{"60 percent", "1234567890", 0.6, "12******90"},
		{"0 percent", "hello", 0, "hello"},
		{"100 percent", "hello", 1.0, "*****"},
		{"empty", "", 0.5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringMiddle(tt.input, tt.maskPercent)
			if result != tt.expected {
				t.Errorf("StringMiddle(%q, %v) = %q, want %q", tt.input, tt.maskPercent, result, tt.expected)
			}
		})
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "13812345678", ModeSimple, "***********"},
		{"pattern", "13812345678", ModePattern, "138****5678"},
		{"empty", "", ModePattern, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Phone(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("Phone(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestIDCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "110101199001011234", ModeSimple, "******************"},
		{"pattern", "110101199001011234", ModePattern, "110***********1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IDCard(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("IDCard(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "user@example.com", ModeSimple, "****************"},
		{"pattern", "user@example.com", ModePattern, "u***@example.com"},
		{"pattern short", "a@b.com", ModePattern, "a@b.com"},
		{"no at", "userexample.com", ModePattern, "u*************m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Email(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("Email(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestBankAccount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "6222021234567890123", ModeSimple, "*******************"},
		{"pattern", "6222021234567890123", ModePattern, "6222***********0123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BankAccount(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("BankAccount(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple 2 chars", "张三", ModeSimple, "**"},
		{"pattern 2 chars", "张三", ModePattern, "张*"},
		{"pattern 3 chars", "张三丰", ModePattern, "张*丰"},
		{"pattern 4 chars", "欧阳震华", ModePattern, "欧**华"},
		{"single char", "张", ModePattern, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Name(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("Name(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "北京市朝阳区xxx街道xxx号", ModeSimple, "***************"},
		{"pattern", "北京市朝阳区xxx街道xxx号", ModePattern, "北京市朝阳区*********"},
		{"short", "北京市", ModePattern, "北京市"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Address(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("Address(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     Mode
		expected string
	}{
		{"simple", "12345.67", ModeSimple, "********"},
		{"pattern", "12345.67", ModePattern, "*****.**"},
		{"pattern with comma", "1,234,567.89", ModePattern, "*,***,***.**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Number(tt.input, tt.mode)
			if result != tt.expected {
				t.Errorf("Number(%q, %v) = %q, want %q", tt.input, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestFloat(t *testing.T) {
	// Test simple mode
	result := Float(12345.67, ModeSimple, 2)
	if result != 0 {
		t.Errorf("MaskFloat simple mode should return 0, got %v", result)
	}

	// Test pattern mode - round to 100
	result = Float(12345.67, ModePattern, 2)
	if result != 12300 {
		t.Errorf("MaskFloat pattern mode should round to 12300, got %v", result)
	}

	// Test statistical mode - should add noise
	result = Float(1000, ModeStatistical, 0)
	if result < 900 || result > 1100 {
		t.Errorf("MaskFloat statistical mode should be within ±10%%, got %v", result)
	}
}

func TestInt(t *testing.T) {
	// Test simple mode
	result := Int(12345, ModeSimple, 100)
	if result != 0 {
		t.Errorf("MaskInt simple mode should return 0, got %v", result)
	}

	// Test pattern mode - round to 100
	result = Int(12345, ModePattern, 100)
	if result != 12300 {
		t.Errorf("MaskInt pattern mode should round to 12300, got %v", result)
	}

	// Test statistical mode
	result = Int(1000, ModeStatistical, 0)
	if result < 900 || result > 1100 {
		t.Errorf("MaskInt statistical mode should be within ±10%%, got %v", result)
	}
}

func TestFloatPreserveStats(t *testing.T) {
	stats := NumericStats{
		Min:    0,
		Max:    100,
		Mean:   50,
		StdDev: 10,
	}

	// Test multiple times to ensure values are within range
	for i := 0; i < 100; i++ {
		result := FloatPreserveStats(50, stats)
		if result < stats.Min || result > stats.Max {
			t.Errorf("MaskFloatPreserveStats result %v outside range [%v, %v]", result, stats.Min, stats.Max)
		}
	}
}

func TestFloatPreserveRange(t *testing.T) {
	min, max := 10.0, 20.0

	for i := 0; i < 100; i++ {
		result := FloatPreserveRange(15, min, max)
		if result < min || result > max {
			t.Errorf("MaskFloatPreserveRange result %v outside range [%v, %v]", result, min, max)
		}
	}
}

func TestByPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"phone", "13812345678", "138****5678"},
		{"email", "user@example.com", "u***@example.com"},
		{"id card", "110101199001011234", "110***********1234"},
		{"bank card", "6222021234567890123", "6222***********0123"},
		{"generic", "hello world", "h*********d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByPattern(tt.input)
			if result != tt.expected {
				t.Errorf("ByPattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithChar(t *testing.T) {
	result := WithChar("13812345678", '#', 3, 4)
	expected := "138####5678"
	if result != expected {
		t.Errorf("MaskWithChar = %q, want %q", result, expected)
	}
}

func TestWithFunc(t *testing.T) {
	// Test custom function
	upper := func(s string) string {
		return "***" + s[len(s)-2:] + "***"
	}

	result := WithFunc("hello", upper)
	if result != "***lo***" {
		t.Errorf("MaskWithFunc = %q, want %q", result, "***lo***")
	}

	// Test nil function
	result = WithFunc("hello", nil)
	if result != "hello" {
		t.Errorf("MaskWithFunc with nil = %q, want %q", result, "hello")
	}
}

// Benchmark tests
func BenchmarkPhone(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Phone("13812345678", ModePattern)
	}
}

func BenchmarkEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Email("user@example.com", ModePattern)
	}
}

func BenchmarkByPattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ByPattern("13812345678")
	}
}

// Example usage
func ExamplePhone() {
	result := Phone("13812345678", ModePattern)
	fmt.Println(result)
	// Output: 138****5678
}

func ExampleEmail() {
	result := Email("user@example.com", ModePattern)
	fmt.Println(result)
	// Output: u***@example.com
}

func TestMaskFloatPreserveStatsZeroStdDev(t *testing.T) {
	stats := NumericStats{
		Min:    50,
		Max:    50,
		Mean:   50,
		StdDev: 0,
	}

	result := FloatPreserveStats(50, stats)
	if math.Abs(result-50) > 0.001 {
		t.Errorf("MaskFloatPreserveStats with zero stddev should return mean, got %v", result)
	}
}

// ==================== Realistic Mode Tests ====================

func TestMaskPhoneRealistic(t *testing.T) {
	phone := "13812345678"
	result := Phone(phone, ModeRealistic)

	// Should be 11 digits
	if len(result) != 11 {
		t.Errorf("MaskPhone realistic should return 11 chars, got %d: %s", len(result), result)
	}

	// Should start with 1
	if result[0] != '1' {
		t.Errorf("MaskPhone realistic should start with 1, got %s", result)
	}

	// Same input should produce same output (deterministic)
	result2 := Phone(phone, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskPhone realistic should be deterministic: %s != %s", result, result2)
	}

	// Different input should produce different output
	result3 := Phone("13999999999", ModeRealistic)
	if result == result3 {
		t.Errorf("MaskPhone realistic should produce different output for different input")
	}
}

func TestMaskIDCardRealistic(t *testing.T) {
	idCard := "110101199001011234"
	result := IDCard(idCard, ModeRealistic)

	// Should be 18 characters
	if len(result) != 18 {
		t.Errorf("MaskIDCard realistic should return 18 chars, got %d: %s", len(result), result)
	}

	// Same input should produce same output (deterministic)
	result2 := IDCard(idCard, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskIDCard realistic should be deterministic: %s != %s", result, result2)
	}

	// Different input should produce different output
	result3 := IDCard("320106199912315678", ModeRealistic)
	if result == result3 {
		t.Errorf("MaskIDCard realistic should produce different output for different input")
	}

	// Verify check digit is valid
	checkDigit := calculateIDCardCheckDigit(result[:17])
	if string(result[17]) != checkDigit {
		t.Errorf("MaskIDCard realistic check digit invalid: expected %s, got %c", checkDigit, result[17])
	}
}

func TestMaskNameRealistic(t *testing.T) {
	name := "张三"
	result := Name(name, ModeRealistic)

	// Should be similar length (2-3 chars for Chinese name)
	resultRunes := []rune(result)
	if len(resultRunes) < 2 || len(resultRunes) > 4 {
		t.Errorf("MaskName realistic should return 2-4 chars, got %d: %s", len(resultRunes), result)
	}

	// Same input should produce same output
	result2 := Name(name, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskName realistic should be deterministic: %s != %s", result, result2)
	}

	// Different input should produce different output
	result3 := Name("李四", ModeRealistic)
	if result == result3 {
		t.Errorf("MaskName realistic should produce different output for different input")
	}
}

func TestEmailRealistic(t *testing.T) {
	email := "user@example.com"
	result := Email(email, ModeRealistic)

	// Should contain @
	if !contains(result, "@") {
		t.Errorf("MaskEmail realistic should contain @: %s", result)
	}

	// Should have domain
	if !contains(result, ".") {
		t.Errorf("MaskEmail realistic should contain domain: %s", result)
	}

	// Same input should produce same output
	result2 := Email(email, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskEmail realistic should be deterministic: %s != %s", result, result2)
	}
}

func TestBankAccountRealistic(t *testing.T) {
	account := "6222021234567890123"
	result := BankAccount(account, ModeRealistic)

	// Should be 16-19 digits
	if len(result) < 16 || len(result) > 19 {
		t.Errorf("MaskBankAccount realistic should return 16-19 chars, got %d: %s", len(result), result)
	}

	// Same input should produce same output
	result2 := BankAccount(account, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskBankAccount realistic should be deterministic: %s != %s", result, result2)
	}
}

func TestAddressRealistic(t *testing.T) {
	address := "北京市朝阳区建国路100号"
	result := Address(address, ModeRealistic)

	// Should not be empty
	if len(result) == 0 {
		t.Errorf("MaskAddress realistic should not be empty")
	}

	// Same input should produce same output
	result2 := Address(address, ModeRealistic)
	if result != result2 {
		t.Errorf("MaskAddress realistic should be deterministic: %s != %s", result, result2)
	}

	// Should contain address-like content
	if !contains(result, "区") && !contains(result, "市") {
		t.Errorf("MaskAddress realistic should look like address: %s", result)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark realistic mode
func BenchmarkMaskPhoneRealistic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Phone("13812345678", ModeRealistic)
	}
}

func BenchmarkMaskIDCardRealistic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IDCard("110101199001011234", ModeRealistic)
	}
}

func BenchmarkMaskNameRealistic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Name("张三", ModeRealistic)
	}
}
