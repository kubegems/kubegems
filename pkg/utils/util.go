package utils

import (
	"errors"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	regA_Z  = regexp.MustCompile("[A-Z]+")
	rega_z  = regexp.MustCompile("[a-z]+")
	reg0_9  = regexp.MustCompile("[0-9]+")
	reg_chs = regexp.MustCompile(`[!\.@#$%~]+`)
)

// 保留n位小数
func RoundTo(n float64, decimals uint32) float64 {
	return math.Round(n*math.Pow(10, float64(decimals))) / math.Pow(10, float64(decimals))
}

// DayStartTime 返回当天0点
func DayStartTime(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func NextDayStartTime(t time.Time) time.Time {
	return DayStartTime(t).Add(24 * time.Hour)
}

func ToUint(id string) uint {
	idint, err := strconv.Atoi(id)
	if err != nil {
		return 0
	}
	return uint(idint)
}

// src中是否存在了dest字符串
func ContainStr(src []string, dest string) bool {
	for i := range src {
		if src[i] == dest {
			return true
		}
	}
	return false
}

func RemoveStrInReplace(src []string, dest string) []string {
	index := 0
	for i := range src {
		if src[i] != dest {
			src[index] = src[i]
			index++
		}
	}
	return src[:index]
}

func RemoveStr(src []string, dest string) []string {
	ret := []string{}
	for i := range src {
		if src[i] != dest {
			ret = append(ret, src[i])
		}
	}
	return ret
}

func ValidPassword(input string) error {
	if len(input) < 8 {
		return errors.New("密码长度至少8位")
	}
	if !regA_Z.Match([]byte(input)) {
		return errors.New("密码至少包含一个大写字母")
	}
	if !rega_z.Match([]byte(input)) {
		return errors.New("密码至少包含一个小写字母")
	}
	if !reg0_9.Match([]byte(input)) {
		return errors.New("密码至少包含一个数字")
	}
	if !reg_chs.Match([]byte(input)) {
		return errors.New("密码至少包含一个特殊字符(.!@#$%~)")
	}

	return nil
}

func MakePassword(input string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func GeneratePassword() string {
	r := []rune{}
	r = append(r, randomRune(4, runeKindLower)...)
	r = append(r, randomRune(3, runeKindUpper)...)
	r = append(r, randomRune(2, runeKindNum)...)
	r = append(r, randomRune(1, runeKindChar)...)
	rand.Shuffle(len(r), func(i, j int) {
		if rand.Intn(10) > 5 {
			r[i], r[j] = r[j], r[i]
		}
	})
	return string(r)
}

func ValidatePassword(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func JoinFlagName(prefix, key string) string {
	if prefix == "" {
		return strings.ToLower(key)
	}
	return strings.ToLower(prefix + "-" + key)
}

func IsValidFQDNLower(s string) bool {
	fq := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	return fq.MatchString(s)
}

const (
	runeKindNum   = "num"
	runeKindLower = "lower"
	runeKindUpper = "upper"
	runeKindChar  = "char"
)

var (
	lowerLetterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	upperLetterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	numRunes         = []rune("0123456789")
	charRunes        = []rune("!.@#$%~")
)

func randomRune(n int, kind string) []rune {
	b := make([]rune, n)
	var l []rune
	switch kind {
	case runeKindChar:
		l = charRunes
	case runeKindUpper:
		l = upperLetterRunes
	case runeKindLower:
		l = lowerLetterRunes
	case runeKindNum:
		l = numRunes
	default:
		l = lowerLetterRunes
	}
	length := len(l)
	for i := range b {
		b[i] = l[rand.Intn(length)]
	}
	return b
}

func BoolToString(a bool) string {
	if a {
		return "1"
	}
	return "0"
}

func BoolToFloat64(a *bool) float64 {
	if a != nil && *a {
		return 1
	}
	return 0
}

func TimeZeroToNull(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	return t
}

func FormatMysqlDumpTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05.000")
}

func UintToStr(i *uint) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(int(*i))
}

func SliceUniqueKey(s []string) string {
	tmp := append([]string{}, s...)
	sort.Strings(tmp)
	return strings.Join(tmp, "-")
}
