package utils

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"errors"
	"math"
	"math/rand"
	"regexp"
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

func StrOrDef(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

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

func ValidPassword(input string) error {
	if len(input) < 8 {
		return errors.New("密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)")
	}
	if !regA_Z.Match([]byte(input)) {
		return errors.New("密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)")
	}
	if !rega_z.Match([]byte(input)) {
		return errors.New("密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)")
	}
	if !reg0_9.Match([]byte(input)) {
		return errors.New("密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)")
	}
	if !reg_chs.Match([]byte(input)) {
		return errors.New("密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)")
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

func BoolToFloat64(a bool) float64 {
	if a {
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

type DesEncryptor struct {
	Key []byte
}

func (e *DesEncryptor) EncryptBase64(input string) (string, error) {
	data := []byte(input)
	block, err := des.NewCipher(e.Key)
	if err != nil {
		return "", err
	}
	data = e.Padding(data, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, e.Key)
	crypted := make([]byte, len(data))
	blockMode.CryptBlocks(crypted, data)
	return base64.StdEncoding.EncodeToString(crypted), nil
}

func (e *DesEncryptor) DecryptBase64(input string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	block, err := des.NewCipher(e.Key)
	if err != nil {
		return "", err
	}
	blockMode := cipher.NewCBCDecrypter(block, e.Key)
	origData := make([]byte, len(data))
	blockMode.CryptBlocks(origData, data)
	origData = e.UnPadding(origData)
	return string(origData), nil
}

func (e *DesEncryptor) Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func (e *DesEncryptor) UnPadding(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return []byte{}
	}
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}
