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
	"crypto/md5"
	"encoding/binary"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"unicode"
)

// Mode defines the masking strategy
type Mode int

const (
	// ModeSimple replaces content with asterisks
	ModeSimple Mode = iota
	// ModePattern preserves format/pattern while masking content
	ModePattern
	// ModeStatistical preserves statistical properties
	ModeStatistical
	// ModeRealistic generates realistic-looking fake data
	// Same input produces same output (deterministic)
	ModeRealistic
)

// DefaultMaskChar is the default masking character
const DefaultMaskChar = '*'

// Common Chinese surnames for realistic name generation
var commonSurnames = []string{
	"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴",
	"徐", "孙", "马", "胡", "朱", "郭", "何", "高", "林", "罗",
	"郑", "梁", "谢", "宋", "唐", "许", "韩", "冯", "邓", "曹",
	"彭", "曾", "萧", "田", "董", "袁", "潘", "于", "蒋", "蔡",
}

// Common Chinese given name characters
var commonGivenChars = []string{
	"伟", "芳", "娜", "秀英", "敏", "静", "丽", "强", "磊", "洋",
	"艳", "勇", "军", "杰", "娟", "涛", "明", "超", "秀兰", "霞",
	"平", "刚", "桂英", "华", "建", "国", "文", "辉", "玲", "婷",
	"宇", "欣", "怡", "博", "浩", "雪", "梅", "鑫", "龙", "凤",
}

// ID card area codes (province prefixes)
var idCardAreaCodes = []string{
	"110101", "110102", "110105", "110106", "110107", "110108", // 北京
	"310101", "310104", "310105", "310106", "310107", "310109", // 上海
	"440103", "440104", "440105", "440106", "440111", "440112", // 广州
	"320102", "320104", "320105", "320106", "320111", "320113", // 南京
	"330102", "330103", "330104", "330105", "330106", "330108", // 杭州
}

// ==================== String Masking ====================

// MaskString masks a string with the given mode
// Simple mode: "hello" -> "*****"
// Pattern mode: "hello" -> "h***o" (keeps first and last char)
func String(s string, mode Mode) string {
	if len(s) == 0 {
		return s
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(s))
	case ModePattern:
		return maskStringPattern(s)
	default:
		return strings.Repeat(string(DefaultMaskChar), len(s))
	}
}

// maskStringPattern preserves first and last character
func maskStringPattern(s string) string {
	runes := []rune(s)
	length := len(runes)

	if length <= 2 {
		return strings.Repeat(string(DefaultMaskChar), length)
	}

	var result strings.Builder
	result.WriteRune(runes[0])
	for i := 1; i < length-1; i++ {
		result.WriteRune(DefaultMaskChar)
	}
	result.WriteRune(runes[length-1])
	return result.String()
}

// MaskStringKeep masks a string while keeping specified number of characters
// keepStart: number of characters to keep at the start
// keepEnd: number of characters to keep at the end
// Example: StringKeep("13812345678", 3, 4) -> "138****5678"
func StringKeep(s string, keepStart, keepEnd int) string {
	runes := []rune(s)
	length := len(runes)

	if length == 0 {
		return s
	}

	if keepStart < 0 {
		keepStart = 0
	}
	if keepEnd < 0 {
		keepEnd = 0
	}

	// If keeping more than total length, return original
	if keepStart+keepEnd >= length {
		return s
	}

	var result strings.Builder
	// Keep start
	for i := 0; i < keepStart; i++ {
		result.WriteRune(runes[i])
	}
	// Mask middle
	maskCount := length - keepStart - keepEnd
	for i := 0; i < maskCount; i++ {
		result.WriteRune(DefaultMaskChar)
	}
	// Keep end
	for i := length - keepEnd; i < length; i++ {
		result.WriteRune(runes[i])
	}

	return result.String()
}

// MaskStringMiddle masks the middle portion of a string by percentage
// maskPercent: percentage of characters to mask (0.0-1.0)
// Example: StringMiddle("1234567890", 0.6) -> "12******90"
func StringMiddle(s string, maskPercent float64) string {
	runes := []rune(s)
	length := len(runes)

	if length == 0 || maskPercent <= 0 {
		return s
	}
	if maskPercent >= 1 {
		return strings.Repeat(string(DefaultMaskChar), length)
	}

	maskCount := int(math.Round(float64(length) * maskPercent))
	if maskCount == 0 {
		return s
	}

	// Calculate start and end positions to keep
	keepTotal := length - maskCount
	keepStart := keepTotal / 2
	keepEnd := keepTotal - keepStart

	return StringKeep(s, keepStart, keepEnd)
}

// ==================== Phone Number Masking ====================

// MaskPhone masks a phone number
// Simple mode: "13812345678" -> "***********"
// Pattern mode: "13812345678" -> "138****5678"
// Realistic mode: "13812345678" -> "13698765432" (looks like real phone)
func Phone(phone string, mode Mode) string {
	if len(phone) == 0 {
		return phone
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(phone))
	case ModePattern:
		// Keep first 3 and last 4 digits
		return StringKeep(phone, 3, 4)
	case ModeRealistic:
		return generateRealisticPhone(phone)
	default:
		return StringKeep(phone, 3, 4)
	}
}

// generateRealisticPhone generates a realistic-looking phone number
// Same input produces same output (deterministic)
func generateRealisticPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}

	// Use hash to generate deterministic but different phone
	hash := hashString(phone)

	// Phone prefixes (Chinese mobile carriers)
	prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139",
		"150", "151", "152", "153", "155", "156", "157", "158", "159",
		"170", "171", "172", "173", "175", "176", "177", "178",
		"180", "181", "182", "183", "184", "185", "186", "187", "188", "189"}

	prefix := prefixes[hash%uint64(len(prefixes))]

	// Generate remaining 8 digits deterministically from hash
	var result strings.Builder
	result.WriteString(prefix)
	for i := 0; i < 8; i++ {
		digit := (hash >> (i * 4)) % 10
		result.WriteByte(byte('0' + digit))
	}

	return result.String()
}

// ==================== ID Card Masking ====================

// MaskIDCard masks an ID card number
// Simple mode: "110101199001011234" -> "******************"
// Pattern mode: "110101199001011234" -> "110***********1234"
// Realistic mode: "110101199001011234" -> "320106198805152345" (looks like real ID)
func IDCard(idCard string, mode Mode) string {
	if len(idCard) == 0 {
		return idCard
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(idCard))
	case ModePattern:
		// Keep first 3 and last 4 digits
		return StringKeep(idCard, 3, 4)
	case ModeRealistic:
		return generateRealisticIDCard(idCard)
	default:
		return StringKeep(idCard, 3, 4)
	}
}

// generateRealisticIDCard generates a realistic-looking ID card number
// Same input produces same output (deterministic) to maintain uniqueness
func generateRealisticIDCard(idCard string) string {
	if len(idCard) != 18 {
		return idCard
	}

	// Use hash to generate deterministic but different ID
	hash := hashString(idCard)

	// Area code (6 digits)
	areaCode := idCardAreaCodes[hash%uint64(len(idCardAreaCodes))]

	// Birth date (8 digits): 1960-2005
	year := 1960 + int((hash>>8)%46)
	month := 1 + int((hash>>12)%12)
	day := 1 + int((hash>>16)%28) // Use 28 to avoid invalid dates

	// Sequence code (3 digits) + check digit
	seq := (hash >> 20) % 999
	if seq < 1 {
		seq = 1
	}

	// Build ID without check digit
	idWithoutCheck := areaCode + intToString(year, 4) + intToString(month, 2) + intToString(day, 2) + intToString(int(seq), 3)

	// Calculate check digit
	checkDigit := calculateIDCardCheckDigit(idWithoutCheck)

	return idWithoutCheck + checkDigit
}

// calculateIDCardCheckDigit calculates the check digit for Chinese ID card
func calculateIDCardCheckDigit(id17 string) string {
	if len(id17) != 17 {
		return "0"
	}

	// Weight factors
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := "10X98765432"

	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(id17[i] - '0')
		sum += digit * weights[i]
	}

	return string(checkCodes[sum%11])
}

// intToString converts int to string with zero padding
func intToString(n int, width int) string {
	s := ""
	for n > 0 || len(s) < width {
		s = string('0'+byte(n%10)) + s
		n /= 10
	}
	if len(s) > width {
		s = s[len(s)-width:]
	}
	return s
}

// ==================== Email Masking ====================

// MaskEmail masks an email address
// Simple mode: "user@example.com" -> "****************"
// Pattern mode: "user@example.com" -> "u***@example.com"
// Realistic mode: "user@example.com" -> "abc123@gmail.com" (looks like real email)
func Email(email string, mode Mode) string {
	if len(email) == 0 {
		return email
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(email))
	case ModePattern:
		atIndex := strings.LastIndex(email, "@")
		if atIndex <= 0 {
			return String(email, ModePattern)
		}
		localPart := email[:atIndex]
		domain := email[atIndex:]

		// Mask local part, keep first char
		if len(localPart) <= 1 {
			return localPart + domain
		}
		maskedLocal := string(localPart[0]) + strings.Repeat(string(DefaultMaskChar), len(localPart)-1)
		return maskedLocal + domain
	case ModeRealistic:
		return EmailRealistic(email)
	default:
		return Email(email, ModePattern)
	}
}

// ==================== Bank Account Masking ====================

// MaskBankAccount masks a bank account number
// Simple mode: "6222021234567890123" -> "*******************"
// Pattern mode: "6222021234567890123" -> "6222***********0123"
// Realistic mode: "6222021234567890123" -> "6217001234567890" (looks like real bank card)
func BankAccount(account string, mode Mode) string {
	if len(account) == 0 {
		return account
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(account))
	case ModePattern:
		// Keep first 4 and last 4 digits
		return StringKeep(account, 4, 4)
	case ModeRealistic:
		return BankAccountRealistic(account)
	default:
		return StringKeep(account, 4, 4)
	}
}

// ==================== Name Masking ====================

// MaskName masks a person's name
// Simple mode: "张三" -> "**"
// Pattern mode: "张三" -> "张*", "张三丰" -> "张*丰"
// Realistic mode: "张三" -> "李明" (looks like real name)
func Name(name string, mode Mode) string {
	runes := []rune(name)
	length := len(runes)

	if length == 0 {
		return name
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), length)
	case ModePattern:
		if length == 1 {
			return string(DefaultMaskChar)
		}
		if length == 2 {
			return string(runes[0]) + string(DefaultMaskChar)
		}
		// Keep first and last character
		var result strings.Builder
		result.WriteRune(runes[0])
		for i := 1; i < length-1; i++ {
			result.WriteRune(DefaultMaskChar)
		}
		result.WriteRune(runes[length-1])
		return result.String()
	case ModeRealistic:
		return generateRealisticName(name)
	default:
		return Name(name, ModePattern)
	}
}

// generateRealisticName generates a realistic-looking Chinese name
// Same input produces same output (deterministic)
func generateRealisticName(name string) string {
	runes := []rune(name)
	length := len(runes)

	if length == 0 {
		return name
	}

	// Use hash to generate deterministic name
	hash := hashString(name)

	surname := commonSurnames[hash%uint64(len(commonSurnames))]
	givenName := commonGivenChars[(hash>>8)%uint64(len(commonGivenChars))]

	// Match original name length
	result := surname + givenName
	resultRunes := []rune(result)

	if len(resultRunes) < length {
		// Add more characters if needed
		extra := commonGivenChars[(hash>>16)%uint64(len(commonGivenChars))]
		result = surname + givenName + extra
		resultRunes = []rune(result)
	}

	if len(resultRunes) > length {
		// Trim to match original length
		result = string(resultRunes[:length])
	}

	return result
}

// ==================== Address Masking ====================

// MaskAddress masks an address
// Simple mode: "北京市朝阳区xxx街道xxx号" -> "***************"
// Pattern mode: "北京市朝阳区xxx街道xxx号" -> "北京市朝阳区******"
// Realistic mode: "北京市朝阳区xxx街道xxx号" -> "上海市浦东新区中山路123号" (looks like real address)
func Address(address string, mode Mode) string {
	runes := []rune(address)
	length := len(runes)

	if length == 0 {
		return address
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), length)
	case ModePattern:
		// Keep first 6 characters (usually province/city/district)
		keepLen := 6
		if length <= keepLen {
			return address
		}
		var result strings.Builder
		for i := 0; i < keepLen; i++ {
			result.WriteRune(runes[i])
		}
		for i := keepLen; i < length; i++ {
			result.WriteRune(DefaultMaskChar)
		}
		return result.String()
	case ModeRealistic:
		return AddressRealistic(address)
	default:
		return Address(address, ModePattern)
	}
}

// ==================== Numeric Masking ====================

// MaskNumber masks a number (as string) while preserving format
// Simple mode: "12345.67" -> "********"
// Pattern mode: "12345.67" -> "*****.##" (preserves decimal structure)
func Number(numStr string, mode Mode) string {
	if len(numStr) == 0 {
		return numStr
	}

	switch mode {
	case ModeSimple:
		return strings.Repeat(string(DefaultMaskChar), len(numStr))
	case ModePattern:
		var result strings.Builder
		for _, r := range numStr {
			if unicode.IsDigit(r) {
				result.WriteRune(DefaultMaskChar)
			} else {
				// Preserve non-digit characters (decimal point, comma, etc.)
				result.WriteRune(r)
			}
		}
		return result.String()
	default:
		return Number(numStr, ModePattern)
	}
}

// MaskFloat masks a float value while optionally preserving statistical properties
// mode: ModeSimple - returns 0, ModePattern - returns rounded value, ModeStatistical - adds noise
// precision: number of decimal places to preserve in pattern mode
func Float(value float64, mode Mode, precision int) float64 {
	switch mode {
	case ModeSimple:
		return 0
	case ModePattern:
		// Round to specified precision to reduce information
		multiplier := math.Pow(10, float64(precision))
		return math.Round(value/multiplier) * multiplier
	case ModeStatistical:
		// Add random noise while preserving approximate magnitude
		noise := (rand.Float64() - 0.5) * 0.2 * value // ±10% noise
		return value + noise
	default:
		return 0
	}
}

// MaskInt masks an integer value
// mode: ModeSimple - returns 0, ModePattern - returns rounded value, ModeStatistical - adds noise
// roundTo: round to nearest multiple (e.g., 10, 100, 1000)
func Int(value int64, mode Mode, roundTo int64) int64 {
	switch mode {
	case ModeSimple:
		return 0
	case ModePattern:
		if roundTo <= 0 {
			roundTo = 1
		}
		return (value / roundTo) * roundTo
	case ModeStatistical:
		// Add random noise while preserving approximate magnitude
		noise := float64(value) * (rand.Float64() - 0.5) * 0.2 // ±10% noise
		return value + int64(noise)
	default:
		return 0
	}
}

// ==================== Statistical Preserving Masking ====================

// NumericStats holds statistical properties of numeric data
type NumericStats struct {
	Min    float64
	Max    float64
	Mean   float64
	StdDev float64
}

// MaskFloatPreserveStats masks a float while preserving statistical properties
// The masked value will be within the same statistical distribution
func FloatPreserveStats(value float64, stats NumericStats) float64 {
	// Normalize the value
	if stats.StdDev == 0 {
		return stats.Mean
	}

	// Calculate z-score
	zScore := (value - stats.Mean) / stats.StdDev

	// Add small random noise to z-score
	noisyZ := zScore + (rand.Float64()-0.5)*0.5

	// Convert back
	result := stats.Mean + noisyZ*stats.StdDev

	// Clamp to range
	if result < stats.Min {
		result = stats.Min
	}
	if result > stats.Max {
		result = stats.Max
	}

	return result
}

// MaskFloatPreserveRange masks a float while keeping it within the original range
func FloatPreserveRange(value float64, min, max float64) float64 {
	if max <= min {
		return min
	}
	// Generate random value in range
	return min + rand.Float64()*(max-min)
}

// ==================== Pattern Detection and Masking ====================

// MaskByPattern masks a string based on detected pattern
// Automatically detects common patterns and applies appropriate masking
func ByPattern(s string) string {
	if len(s) == 0 {
		return s
	}

	// Detect phone number pattern
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if phoneRegex.MatchString(s) {
		return Phone(s, ModePattern)
	}

	// Detect email pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if emailRegex.MatchString(s) {
		return Email(s, ModePattern)
	}

	// Detect ID card pattern (18 digits, last may be X)
	idCardRegex := regexp.MustCompile(`^\d{17}[\dXx]$`)
	if idCardRegex.MatchString(s) {
		return IDCard(s, ModePattern)
	}

	// Detect bank card pattern (16-19 digits)
	bankCardRegex := regexp.MustCompile(`^\d{16,19}$`)
	if bankCardRegex.MatchString(s) {
		return BankAccount(s, ModePattern)
	}

	// Default: use pattern masking for strings
	return String(s, ModePattern)
}

// ==================== Custom Masking ====================

// MaskWithChar masks a string using a custom masking character
func WithChar(s string, maskChar rune, keepStart, keepEnd int) string {
	runes := []rune(s)
	length := len(runes)

	if length == 0 {
		return s
	}

	if keepStart < 0 {
		keepStart = 0
	}
	if keepEnd < 0 {
		keepEnd = 0
	}

	if keepStart+keepEnd >= length {
		return s
	}

	var result strings.Builder
	for i := 0; i < keepStart; i++ {
		result.WriteRune(runes[i])
	}
	for i := keepStart; i < length-keepEnd; i++ {
		result.WriteRune(maskChar)
	}
	for i := length - keepEnd; i < length; i++ {
		result.WriteRune(runes[i])
	}

	return result.String()
}

// Func is a function type for custom masking logic
type Func func(s string) string

// MaskWithFunc applies a custom masking function
func WithFunc(s string, fn Func) string {
	if fn == nil {
		return s
	}
	return fn(s)
}

// ==================== Hash Utility ====================

// hashString generates a deterministic hash for a string
// Same input always produces same output
func hashString(s string) uint64 {
	h := md5.Sum([]byte(s))
	return binary.BigEndian.Uint64(h[:8])
}

// ==================== Email Realistic Mode ====================

// Common email providers for realistic email generation
var emailProviders = []string{
	"gmail.com", "yahoo.com", "hotmail.com", "outlook.com",
	"163.com", "126.com", "qq.com", "sina.com", "sohu.com",
}

// MaskEmailRealistic generates a realistic-looking email
func EmailRealistic(email string) string {
	if len(email) == 0 {
		return email
	}

	hash := hashString(email)

	// Generate username part
	userLen := 6 + int(hash%6) // 6-11 characters
	var user strings.Builder
	for i := 0; i < userLen; i++ {
		char := 'a' + byte((hash>>(i*4))%26)
		user.WriteByte(char)
	}

	// Select domain
	domain := emailProviders[(hash>>32)%uint64(len(emailProviders))]

	return user.String() + "@" + domain
}

// ==================== Bank Account Realistic Mode ====================

// Bank card prefixes (BIN numbers)
var bankCardPrefixes = []string{
	"621700", "621698", "621799", // ICBC
	"622202", "622203", "955880", // ICBC credit
	"621660", "621662", "621663", // ABC
	"622848", "622849", "620061", // BOC
	"621225", "621226", "621468", // CCB
	"622580", "622588", "622598", // CMB
}

// MaskBankAccountRealistic generates a realistic-looking bank account
func BankAccountRealistic(account string) string {
	if len(account) == 0 {
		return account
	}

	hash := hashString(account)

	// Select bank prefix
	prefix := bankCardPrefixes[hash%uint64(len(bankCardPrefixes))]

	// Generate remaining digits (total 16-19 digits)
	totalLen := 16 + int(hash%4)
	remaining := totalLen - len(prefix)

	var result strings.Builder
	result.WriteString(prefix)
	for i := 0; i < remaining; i++ {
		digit := (hash >> (i * 4)) % 10
		result.WriteByte(byte('0' + digit))
	}

	return result.String()
}

// ==================== Address Realistic Mode ====================

// Address templates for realistic address generation
var provinces = []string{"北京市", "上海市", "广东省", "江苏省", "浙江省", "山东省", "四川省", "湖北省"}
var districts = []string{"朝阳区", "海淀区", "东城区", "西城区", "浦东新区", "黄浦区", "天河区", "越秀区"}
var streets = []string{"中山路", "人民路", "解放路", "建设路", "和平路", "文化路", "长安街", "南京路"}

// MaskAddressRealistic generates a realistic-looking address
func AddressRealistic(address string) string {
	if len(address) == 0 {
		return address
	}

	hash := hashString(address)

	province := provinces[hash%uint64(len(provinces))]
	district := districts[(hash>>8)%uint64(len(districts))]
	street := streets[(hash>>16)%uint64(len(streets))]
	number := 1 + int((hash>>24)%999)
	unit := 1 + int((hash>>32)%20)
	room := 101 + int((hash>>40)%899)

	return province + district + street + intToString(number, 1) + "号" + intToString(unit, 1) + "单元" + intToString(room, 1) + "室"
}
