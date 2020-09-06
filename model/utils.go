package model

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pborman/uuid"
)

const (
	LOWERCASE_LETTERS     = "abcdefghijklmnopqrstuvwxyz"
	UPPERCASE_LETTERS     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	NUMBERS               = "0123456789"
	SYMBOLS               = " !\"\\#$%&'()*+,-./:;<=>?@[]^_`|~"
	MAX_PARSE_TAG_COUNT   = 5
	MAX_PARSE_REPLY_COUNT = 3
)

type StringInterface map[string]interface{}
type StringMap map[string]string
type StringArray []string

func (sa StringArray) Equals(input StringArray) bool {

	if len(sa) != len(input) {
		return false
	}

	for index := range sa {
		if sa[index] != input[index] {
			return false
		}
	}

	return true
}

type AppError struct {
	Id            string `json:"id"`
	Message       string `json:"message"` // for users
	DetailedError string `json:"detailed_error"`
	RequestId     string `json:"request_id,omitempty"` // same value of HEADER_REQUEST_ID header
	StatusCode    int    `json:"status_code,omitempty"`
	Where         string `json:"-"`
	params        map[string]interface{}
}

func (er *AppError) Error() string {
	return er.Where + ": " + er.Message + ", " + er.DetailedError
}

func (er *AppError) Translate() {
	// TODO: i18n
	if er.params == nil {
		er.Message = er.Id
	} else {
		er.Message = er.Id + " " + StringInterfaceToJson(er.params)
	}
}

func (er *AppError) ToJson() string {
	b, _ := json.Marshal(er)
	return string(b)
}

func NewAppError(where string, id string, params map[string]interface{}, details string, status int) *AppError {
	ap := &AppError{}
	ap.Id = id
	ap.params = params
	ap.Message = id
	ap.Where = where
	ap.DetailedError = details
	ap.StatusCode = status

	ap.Translate()

	return ap
}

// MapToJson converts a map to a json string
func MapToJson(objmap map[string]string) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

// MapBoolToJson converts a map to a json string
func MapBoolToJson(objmap map[string]bool) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

// MapFromJson will decode the key/value pair map
func MapFromJson(data io.Reader) map[string]string {
	decoder := json.NewDecoder(data)

	var objmap map[string]string
	if err := decoder.Decode(&objmap); err != nil {
		return make(map[string]string)
	} else {
		return objmap
	}
}

// MapFromJson will decode the key/value pair map
func MapBoolFromJson(data io.Reader) map[string]bool {
	decoder := json.NewDecoder(data)

	var objmap map[string]bool
	if err := decoder.Decode(&objmap); err != nil {
		return make(map[string]bool)
	} else {
		return objmap
	}
}

func ArrayToJson(objmap []string) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

func ArrayFromJson(data io.Reader) []string {
	decoder := json.NewDecoder(data)

	var objmap []string
	if err := decoder.Decode(&objmap); err != nil {
		return make([]string, 0)
	} else {
		return objmap
	}
}

func ArrayFromInterface(data interface{}) []string {
	stringArray := []string{}

	dataArray, ok := data.([]interface{})
	if !ok {
		return stringArray
	}

	for _, v := range dataArray {
		if str, ok := v.(string); ok {
			stringArray = append(stringArray, str)
		}
	}

	return stringArray
}

func StringInterfaceToJson(objmap map[string]interface{}) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

func StringInterfaceFromJson(data io.Reader) map[string]interface{} {
	decoder := json.NewDecoder(data)

	var objmap map[string]interface{}
	if err := decoder.Decode(&objmap); err != nil {
		return make(map[string]interface{})
	} else {
		return objmap
	}
}

func StringToJson(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func StringFromJson(data io.Reader) string {
	decoder := json.NewDecoder(data)

	var s string
	if err := decoder.Decode(&s); err != nil {
		return ""
	} else {
		return s
	}
}

func IsValidId(value string) bool {
	if len(value) != 26 {
		return false
	}

	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}

	return true
}

// Copied from https://golang.org/src/net/dnsclient.go#L119
func IsDomainName(s string) bool {
	// See RFC 1035, RFC 3696.
	// Presentation format has dots before every label except the first, and the
	// terminal empty label is optional here because we assume fully-qualified
	// (absolute) input. We must therefore reserve space for the first and last
	// labels' length octets in wire format, where they are necessary and the
	// maximum total length is 255.
	// So our _effective_ maximum is 253, but 254 is not rejected if the last
	// character is a dot.
	l := len(s)
	if l == 0 || l > 254 || l == 254 && s[l-1] != '.' {
		return false
	}

	last := byte('.')
	ok := false // Ok once we've seen a letter.
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_':
			ok = true
			partlen++
		case '0' <= c && c <= '9':
			// fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}
			partlen++
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}

	return ok
}

var encoding = base32.NewEncoding("ykvz9hnx81arj3dml762oq0wgi5pcube")

func NewId() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26)
	return b.String()
}

func NewRandomString(length int) string {
	var b bytes.Buffer
	str := make([]byte, length+8)
	rand.Read(str)
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(str)
	encoder.Close()
	b.Truncate(length)
	return b.String()
}

func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func GetMillisForTime(thisTime time.Time) int64 {
	return thisTime.UnixNano() / int64(time.Millisecond)
}

// get milliseconds since epoch for provided date's start of day
func GetStartOfDayMillis(thisTime time.Time, timeZoneOffset int) int64 {
	localSearchTimeZone := time.FixedZone("Local Search Time Zone", timeZoneOffset)
	resultTime := time.Date(thisTime.Year(), thisTime.Month(), thisTime.Day(), 0, 0, 0, 0, localSearchTimeZone)
	return GetMillisForTime(resultTime)
}

// get milliseconds since epoch for provided date's end of day
func GetEndOfDayMillis(thisTime time.Time, timeZoneOffset int) int64 {
	localSearchTimeZone := time.FixedZone("Local Search Time Zone", timeZoneOffset)
	resultTime := time.Date(thisTime.Year(), thisTime.Month(), thisTime.Day(), 23, 59, 59, 999999999, localSearchTimeZone)
	return GetMillisForTime(resultTime)
}

// pad 2 digit date parts with zeros to meet ISO 8601 format
func PadDateStringZeros(dateString string) string {
	parts := strings.Split(dateString, "-")
	for index, part := range parts {
		if len(part) == 1 {
			parts[index] = "0" + part
		}
	}
	dateString = strings.Join(parts[:], "-")
	return dateString
}

func IsLower(s string) bool {
	return strings.ToLower(s) == s
}

func IsValidEmail(email string) bool {
	if !IsLower(email) {
		return false
	}

	if addr, err := mail.ParseAddress(email); err != nil {
		return false
	} else if addr.Name != "" {
		return false
	}

	return true
}

var tagStart = regexp.MustCompile(`^#{1,}`)
var puncStart = regexp.MustCompile(`^[^\pL\d]+`)
var puncEnd = regexp.MustCompile(`[^\pL\d]+$`)

var validTag = regexp.MustCompile(`^(\pL[\pL\d\_]*[\pL\d])$`)
var validSharpTag = regexp.MustCompile(`^(#\pL[\pL\d\_]*[\pL\d])$`)

func ParseTags(text string) string {
	words := strings.Fields(text)
	m := make(map[string]bool)
	var uniq []string

	for _, word := range words {
		word = puncStart.ReplaceAllString(word, "")
		word = puncEnd.ReplaceAllString(word, "")
		if !m[word] {
			m[word] = true
			uniq = append(uniq, word)
		}
	}

	tagString := ""
	cnt := 0
	for _, word := range uniq {
		if cnt >= MAX_PARSE_TAG_COUNT {
			break
		}

		if len(word) >= TAG_MIN_RUNES && len(word) <= TAG_MAX_RUNES && validTag.MatchString(word) {
			tagString += " " + word
			cnt++
		}
	}

	return strings.TrimSpace(tagString)
}

var validReplies = regexp.MustCompile(`^@[a-z0-9\.\-_]+$`)
var replyStart = regexp.MustCompile(`^@{1,}`)

func ParseReplies(text string) []string {
	m := make(map[string]bool)
	uniqNames := []string{}

	words := strings.Fields(text)
	for _, word := range words {
		word = replyStart.ReplaceAllString(word, "@")
		if validReplies.MatchString(word) {
			word = replyStart.ReplaceAllString(word, "")

			if !m[word] {
				m[word] = true
				uniqNames = append(uniqNames, word)
			}

			if len(uniqNames) >= MAX_PARSE_REPLY_COUNT {
				break
			}
		}
	}

	return uniqNames
}

// AppErrorFromJson will decode the input and return an AppError
func AppErrorFromJson(data io.Reader) *AppError {
	str := ""
	bytes, rerr := ioutil.ReadAll(data)
	if rerr != nil {
		str = rerr.Error()
	} else {
		str = string(bytes)
	}

	decoder := json.NewDecoder(strings.NewReader(str))
	var er AppError
	err := decoder.Decode(&er)
	if err == nil {
		return &er
	} else {
		return NewAppError("AppErrorFromJson", "model.utils.decode_json.app_error", nil, "body: "+str, http.StatusInternalServerError)
	}
}

func CopyStringMap(originalMap map[string]string) map[string]string {
	copyMap := make(map[string]string)
	for k, v := range originalMap {
		copyMap[k] = v
	}
	return copyMap
}

func IsValidAlphaNum(s string) bool {
	validAlphaNum := regexp.MustCompile(`^[a-z0-9]+([a-z\-0-9]+|(__)?)[a-z0-9]+$`)

	return validAlphaNum.MatchString(s)
}

// SanitizeUnicode will remove undesirable Unicode characters from a string.
func SanitizeUnicode(s string) string {
	return strings.Map(filterBlacklist, s)
}

// filterBlacklist returns `r` if it is not in the blacklist, otherwise drop (-1).
// Blacklist is taken from https://www.w3.org/TR/unicode-xml/#Charlist
func filterBlacklist(r rune) rune {
	const drop = -1
	switch r {
	case '\u0340', '\u0341': // clones of grave and acute; deprecated in Unicode
		return drop
	case '\u17A3', '\u17D3': // obsolete characters for Khmer; deprecated in Unicode
		return drop
	case '\u2028', '\u2029': // line and paragraph separator
		return drop
	case '\u202A', '\u202B', '\u202C', '\u202D', '\u202E': // BIDI embedding controls
		return drop
	case '\u206A', '\u206B': // activate/inhibit symmetric swapping; deprecated in Unicode
		return drop
	case '\u206C', '\u206D': // activate/inhibit Arabic form shaping; deprecated in Unicode
		return drop
	case '\u206E', '\u206F': // activate/inhibit national digit shapes; deprecated in Unicode
		return drop
	case '\uFFF9', '\uFFFA', '\uFFFB': // interlinear annotation characters
		return drop
	case '\uFEFF': // byte order mark
		return drop
	case '\uFFFC': // object replacement character
		return drop
	}

	// Scoping for musical notation
	if r >= 0x0001D173 && r <= 0x0001D17A {
		return drop
	}

	// Language tag code points
	if r >= 0x000E0000 && r <= 0x000E007F {
		return drop
	}

	return r
}

func IsValidGroupIdentifier(s string) bool {
	if !IsValidAlphaNumHyphenUnderscore(s) {
		return false
	}

	if len(s) < GROUP_NAME_MIN_LENGTH {
		return false
	}

	return true
}

func IsValidAlphaNumHyphenUnderscore(s string) bool {
	validSimpleAlphaNumHyphenUnderscore := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	return validSimpleAlphaNumHyphenUnderscore.MatchString(s)
}

func IsValidNumberString(value string) bool {
	if _, err := strconv.Atoi(value); err != nil {
		return false
	}

	return true
}

func IsValidHttpUrl(rawUrl string) bool {
	if strings.Index(rawUrl, "http://") != 0 && strings.Index(rawUrl, "https://") != 0 {
		return false
	}

	if _, err := url.ParseRequestURI(rawUrl); err != nil {
		return false
	}

	return true
}
