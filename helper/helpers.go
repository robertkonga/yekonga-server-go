package helper

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
	"github.com/robertkonga/yekonga-server-go/plugins/uuid"
)

func IsMap(data interface{}) bool {
	// Type assertion to check if data is a map[string]interface{}
	if _, ok := data.(map[string]interface{}); ok {
		return true
	} else if _, ok := data.(datatype.Record); ok {
		return true
	} else if _, ok := data.(datatype.DataMap); ok {
		return true
	} else if _, ok := data.(datatype.Context); ok {
		return true
	} else if _, ok := data.(datatype.ContextObject); ok {
		return true
	} else if _, ok := data.(datatype.JsonObject); ok {
		return true
	} else if _, ok := data.(bson.M); ok {
		return true
	}

	return false
}

func ToInt(value interface{}) int {
	number := 0
	if v, ok := value.(string); ok {
		n, err := strconv.Atoi(v)
		if err == nil {
			number = n
		}
	} else if v, ok := value.(int); ok {
		number = v
	}

	return number
}

func ToFloat(value interface{}) float64 {
	var number float64 = 0
	if v, ok := value.(string); ok {
		n, err := strconv.ParseFloat(v, 64)
		if err == nil {
			number = n
		}
	} else if v, ok := value.(float64); ok {
		number = v
	}

	return number
}

func ToFloat64(value interface{}) float64 {
	return ToFloat(value)
}

func CompareValues(a, b interface{}) int {
	// Convert both values to float64 for comparison
	aVal := ToFloat64(a)
	bVal := ToFloat64(b)

	if aVal < bVal {
		return -1
	} else if aVal > bVal {
		return 1
	}
	return 0
}

func ToJson(data interface{}) string {
	jsonData, _ := json.MarshalIndent(data, "", " ")

	return string(jsonData)
}

func ToByte(data interface{}) []byte {
	jsonData, _ := json.Marshal(data)

	return jsonData
}

// JSON file and converts it to a map
func ToMap[T any](data interface{}) map[string]T {
	var result map[string]T
	if err := json.Unmarshal(ToByte(data), &result); err != nil {
		return nil
	}

	return result
}

func ToMapList[T any](data interface{}) []map[string]T {
	converted := []map[string]T{}
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		count := val.Len()
		converted = make([]map[string]T, count)

		for i := 0; i < count; i++ {
			elem := ToMap[T](val.Index(i).Interface())
			converted[i] = elem
		}
	}

	return converted
}

func ToList[T any](data interface{}) []T {
	converted := []T{}
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		count := val.Len()
		converted = make([]T, 0, count)

		for i := 0; i < count; i++ {
			var result T
			if err := json.Unmarshal(ToByte(val.Index(i).Interface()), &result); err != nil {
				console.Error("ToList", err.Error())
			} else {
				converted = append(converted, result)
			}
		}

	}

	return converted
}

func ToInterface(data interface{}) (interface{}, error) {
	var result interface{}

	// JSON file and converts it to a map
	if err := json.Unmarshal(ToByte(data), &result); err != nil {
		console.Error("ConvertTo", err.Error())
		return result, err
	}

	return result, nil
}

func UUID() string {
	id, _ := uuid.NewV1()

	return id.String()
}

func IsSlice(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

func IsList(v interface{}) bool {
	return IsSlice(v)
}

func IsArray(v interface{}) bool {
	return IsSlice(v)
}

func IsUsernameIdentifier() bool {
	return (len(config.Config.UserIdentifiers) == 0 || Contains(config.Config.UserIdentifiers, "username"))
}

func IsPhoneIdentifier() bool {
	return (len(config.Config.UserIdentifiers) == 0 || Contains(config.Config.UserIdentifiers, "phone"))
}

func IsEmailIdentifier() bool {
	return (len(config.Config.UserIdentifiers) == 0 || Contains(config.Config.UserIdentifiers, "email"))
}

func IsWhatsappIdentifier() bool {
	return (len(config.Config.UserIdentifiers) == 0 || Contains(config.Config.UserIdentifiers, "whatsapp"))
}

// convertTo converts a map[string]interface{} to a struct of type T
func ConvertTo[T any](data interface{}) (T, error) {
	var result T

	// JSON file and converts it to a map
	if err := json.Unmarshal(ToByte(data), &result); err != nil {
		console.Error("ConvertTo", err.Error())
		return result, err
	}

	return result, nil
}

// convertTo converts a map[string]interface{} to a struct of type T
func ConvertToDataMap(data map[string]interface{}) datatype.DataMap {
	var result datatype.DataMap

	// Get the reflect.Value of the result struct
	val := reflect.ValueOf(&result).Elem()
	if !val.CanSet() {
		return result
	}

	// Ensure the result is a struct
	if val.Kind() != reflect.Struct {
		return result
	}

	// Iterate over the struct fields
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name

		// Look for a matching key in the map (case-sensitive)
		mapValue, exists := data[fieldName]
		if !exists {
			continue // Skip if the map doesn't have this field
		}

		// Get the reflect.Value of the field
		fieldVal := val.Field(i)
		if !fieldVal.CanSet() {
			continue // Skip if the field is unexported
		}

		// Convert the map value to the field's type
		if err := setField(fieldVal, mapValue); err != nil {
			return result
		}
	}

	return result
}

// setField sets the reflect.Value of a struct field to the provided value
func setField(field reflect.Value, value interface{}) error {
	// Convert the value to the field's type
	val := reflect.ValueOf(value)

	// Check if the value can be converted to the field's type
	if !val.Type().ConvertibleTo(field.Type()) {
		return fmt.Errorf("cannot convert %v to %v", val.Type(), field.Type())
	}

	// Perform the conversion and set the field
	field.Set(val.Convert(field.Type()))
	return nil
}

func IsEmpty(value interface{}) bool {
	if IsPointer(value) {
		v := reflect.ValueOf(value).Elem()

		if v.IsValid() {
			value = v.Interface()
		} else {
			value = nil
		}
	}

	if value == nil || value == "" {
		return true
	} else if v, ok := value.(string); ok && strings.TrimSpace(v) == "" {
		return true
	} else if v, ok := value.([]any); ok && len(v) == 0 {
		return true
	} else if v, ok := value.(map[string]interface{}); ok && len(v) == 0 {
		return true
	}

	return false
}

func IsNotEmpty(value interface{}) bool {
	return !IsEmpty(value)
}

// Utility function to check if slice contains an element
func Contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func Reverse[T interface{}](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func SortMap[T interface{}](options map[string]T) map[string]T {
	// 1. Get all keys into a slice
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}

	// 2. Sort the keys based on the map values
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Compare(ToString(options[a]), ToString(options[b]))
	})

	// 3. Print the results in order
	fmt.Println("Sorted by Value:", keys)
	newOptions := make(map[string]T)
	for _, k := range keys {
		fmt.Printf("%s: %s\n", k, options[k])
		newOptions[k] = options[k]
	}

	return newOptions
}

func SortedKeys[T interface{}](options map[string]T) []string {
	// 1. Get all keys into a slice
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}

	// 2. Sort the keys based on the map values
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Compare(ToString(options[a]), ToString(options[b]))
	})

	return keys
}

// ToCamelCase converts a string to CamelCase
func ToCamelCase(s string) string {
	s = ToUnderscore(s)
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	for i := range words {
		words[i] = strings.Title(strings.ToLower(words[i]))
	}

	return strings.Join(words, "")
}

// ToSlug converts a string to a URL-friendly slug
func ToSlug(s string) string {
	// Convert to lowercase
	s = ToUnderscore(s)

	// Replace non-alphanumeric characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")

	// Trim leading and trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

func ToString(s interface{}) string {
	return fmt.Sprintf("%v", s)
}

// camelToSnake converts camelCase or PascalCase to snake_case
func CamelToSnake(s string) string {
	var (
		result    []rune
		prevLower bool
		prevUpper bool
		prevDigit bool
	)
	for _, r := range s {

		isLower := unicode.IsLower(r)
		isUpper := unicode.IsUpper(r)
		isDigit := unicode.IsDigit(r)

		if (prevLower && isUpper) ||
			(prevDigit && (isLower || isUpper)) ||
			(isDigit && (prevLower || prevUpper)) {
			result = append(result, '_')
		}

		// if isUpper {
		// 	// Add underscore before uppercase letter (except first char)
		// 	result = append(result, '_')
		// }
		result = append(result, unicode.ToLower(r))

		prevLower = isLower
		prevUpper = isUpper
		prevDigit = isDigit
	}

	if len(result) == 0 {
		_ = fmt.Sprint(prevDigit, prevLower, prevUpper)
	}

	return string(result)
}

// ToUnderscore converts a string into snake_case format.
// It handles camelCase, PascalCase, and kebab-case.
func ToUnderscore(text string) string {
	if text == "" {
		return ""
	}

	// 1. Insert underscore before capital letters (camelCase/PascalCase support)
	t := CamelToSnake(text)

	// 2. Convert to lowercase
	t = strings.ToLower(t)

	// 3. Replace spaces and hyphens with a single underscore
	// Example: "hello-world thing" -> "hello_world_thing"
	reSeparator := regexp.MustCompile("[\\s-]+")
	t = reSeparator.ReplaceAllString(t, "_")

	// 4. Remove any multiple consecutive underscores resulting from the previous steps
	reMultipleUnderscores := regexp.MustCompile("_+")
	t = reMultipleUnderscores.ReplaceAllString(t, "_")

	// 5. Remove leading and trailing underscores
	t = strings.Trim(t, "_")

	return t
}

func ToVariable(s string) string {
	s = ToCamelCase(s)

	s = strings.ToLower(s[0:1]) + s[1:]

	return s
}

func ToTitle(s string) string {
	if s == "" {
		return ""
	}

	// Step 1: camelCase → snake_case
	s = ToUnderscore(s)

	// Step 2: collapse multiple underscores → single space
	re := regexp.MustCompile(`_+`)
	s = re.ReplaceAllString(s, " ")

	// Step 3: trim and split into words
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	words := strings.Split(s, " ")
	var titleWords []string

	for _, word := range words {
		if word == "" {
			continue
		}
		// Title case: first rune uppercase, rest unchanged
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		titleWords = append(titleWords, string(runes))
	}

	return strings.Join(titleWords, " ")
}

// Pluralization rules
var pluralRules = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile("(s|sh|ch|x|z)$"), "${1}es"}, // e.g., bus → buses, box → boxes
	{regexp.MustCompile("([^aeiou])y$"), "${1}ies"},  // e.g., city → cities
	{regexp.MustCompile("(f|fe)$"), "ves"},           // e.g., knife → knives
	{regexp.MustCompile("$"), "s"},                   // Default rule: add "s"
}

// Singularization rules
var singularRules = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile("ies$"), "y"},                 // e.g., cities → city
	{regexp.MustCompile("ves$"), "f"},                 // e.g., knives → knife
	{regexp.MustCompile("(s|sh|ch|x|z)$es$"), "${1}"}, // e.g., boxes → box
	{regexp.MustCompile("s$"), ""},                    // Default rule: remove "s"
}

// Pluralize converts a singular noun to its plural form
func Pluralize(word string) string {
	word = Singularize(word)

	for _, rule := range pluralRules {
		if rule.pattern.MatchString(word) {
			return rule.pattern.ReplaceAllString(word, rule.replace)
		}
	}
	return word
}

// Singularize converts a plural noun to its singular form
func Singularize(word string) string {
	for _, rule := range singularRules {
		if rule.pattern.MatchString(word) {
			return rule.pattern.ReplaceAllString(word, rule.replace)
		}
	}
	return word
}

// LoadFile reads a JSON file and converts it to a map
func LoadFile(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return ""
	}

	return string(bytes)
}

// LoadJSONFile reads a JSON file and converts it to a map
func LoadJSONFile(filename string) (map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func GetClientIP(req *http.Request) string {
	ip := req.Header.Get("x-forwarded-for")

	if IsEmpty(ip) {
		ip = req.Header.Get("x-real-ip")
	}

	if IsNotEmpty(ip) {
		ip = strings.Split(ip, ":")[0]
	}

	return ip
}

// GetLocalIP retrieves the first non-loopback local IP address
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil { // Only return IPv4
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no local IP found")
}

// GetLocalIP retrieves the first non-loopback local IP address
func GetLocalIPS() ([]string, error) {
	ips := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() != nil { // Only return IPv4
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	if len(ips) > 0 {
		return ips, nil
	}

	return nil, fmt.Errorf("no local IP found")
}

// GetPublicIP fetches the external IP address from an API
func GetPublicIP() (string, error) {
	resp, err := http.Get("https://api64.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

// GetNetworkIP retrieves the local IP and calculates the network address
func GetNetworkIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue // Ignore down and loopback interfaces
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil { // Only consider IPv4
					networkIP := ipNet.IP.Mask(ipNet.Mask) // Calculate network address
					return networkIP.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no network IP found")
}

// writeCounter tracks download progress
type writeCounter struct {
	downloaded int64
	total      int64
	progress   func(downloaded, total int64)
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.downloaded += int64(n)
	if wc.progress != nil {
		wc.progress(wc.downloaded, wc.total)
	}
	return n, nil
}

// DownloadFile downloads a file from URL and saves it to destPath
// Supports:
//   - Large files (streaming)
//   - Progress callback
//   - Timeout
//   - Resume (optional, see note below)
//   - Automatic directory creation
func DownloadFile(url, destPath string, progress func(downloaded, total int64)) error {
	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Adjust as needed
	}

	// HEAD request to get file size (optional but useful)
	var totalSize int64
	headResp, err := client.Head(url)
	if err != nil {
		return fmt.Errorf("HEAD request failed: %w", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEAD request failed with status: %s", headResp.Status)
	}

	if cl := headResp.Header.Get("Content-Length"); cl != "" {
		if size, err := strconv.ParseInt(cl, 10, 64); err == nil {
			totalSize = size
		}
	}

	// GET request to download
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create output file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Stream with progress
	counter := &writeCounter{
		total:    totalSize,
		progress: progress,
	}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)

	if err != nil {
		console.Error("FileExists", err)
	}

	return !os.IsNotExist(err)
}

// IsNumeric checks if a value is an int, float, or a numeric string
func IsNumeric(value interface{}) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true // Directly numeric types

	case string:
		_, err := strconv.ParseFloat(v, 64)
		return err == nil // Returns true if string is a valid number

	default:
		return false // Not numeric
	}
}

func ConvertCalculatedValue(value interface{}) interface{} {
	if IsNumeric(value) {
		if v, ok := value.(string); ok {
			vi, _ := strconv.ParseFloat(v, 64)
			return vi
		} else {
			return v
		}
	}

	return GetTimestamp(value)
}

func GetTimestamp(value interface{}) time.Time {
	result := StringToDatetime(value)

	if result != nil {
		return (*result).UTC()
	}

	return time.Now().UTC()
}

func ToTimestampString(value interface{}, layout string) time.Time {
	if IsEmpty(layout) {
		layout = "2006"
	}
	if v, ok := value.(string); ok {
		parsedTime, _ := time.Parse(layout, v)

		return parsedTime.UTC()
	} else if v, ok := value.(time.Time); ok {
		return v
	}

	return time.Now().UTC()
}

func StringToDatetime(value interface{}) *time.Time {
	if strValue, ok := value.(string); ok {
		var t time.Time
		var err error

		t, err = time.Parse(time.DateOnly, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.DateTime, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.UnixDate, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.RFC3339, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.RFC822, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.TimeOnly, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.RFC850, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.UnixDate, strValue)
		if err == nil {
			return &t
		}

		ISO_8601 := "2006-01-02T15:04:05Z07:00"
		RFC_1123Z := "Tue Dec 30 2025 11:00:59 GMT+0300 (East Africa Time)"

		t, err = time.Parse(ISO_8601, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(RFC_1123Z, strValue)
		if err == nil {
			return &t
		}

	} else if strValue, ok := value.(time.Time); ok {
		return &strValue
	} else if strValue, ok := value.(bson.DateTime); ok {
		t := strValue.Time()

		return &t
	}

	return nil
}

func StringToTimeOnly(value interface{}) *time.Time {
	if strValue, ok := value.(string); ok {
		var t time.Time
		var err error

		t, err = time.Parse(time.TimeOnly, strValue)
		if err == nil {
			return &t
		}

		t, err = time.Parse(time.Kitchen, strValue)
		if err == nil {
			return &t
		}

	} else if strValue, ok := value.(time.Time); ok {
		return &strValue
	}

	return nil
}

func Yesterday() time.Time {
	t := time.Now().Add(time.Hour * -24)
	result := StringToDatetime(t.Format(time.DateOnly) + " 00:00:00")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func YesterdayEnd() time.Time {
	t := time.Now().Add(time.Hour * -24)
	result := StringToDatetime(t.Format(time.DateOnly) + " 23:59:59")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func Today() time.Time {
	t := time.Now()
	result := StringToDatetime(t.Format(time.DateOnly) + " 00:00:00")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func TodayEnd() time.Time {
	t := time.Now()
	result := StringToDatetime(t.Format(time.DateOnly) + " 23:59:59")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func Tomorrow() time.Time {
	t := time.Now().Add(time.Hour * 24)
	result := StringToDatetime(t.Format(time.DateOnly) + " 00:00:00")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func TomorrowEnd() time.Time {
	t := time.Now().Add(time.Hour * 24)
	result := StringToDatetime(t.Format(time.DateOnly) + " 23:59:59")
	if result != nil {
		return *result
	}

	return t.UTC()
}

func DateStart(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	result := StringToDatetime(t.Format(time.DateOnly) + " 00:00:00")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func DateEnd(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}
	result := StringToDatetime(t.Format(time.DateOnly) + " 23:59:59")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func HourStart(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	format := "2006-01-02 15"
	result := StringToDatetime(t.Format(format) + ":00:00")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func HourEnd(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	format := "2006-01-02 15"
	result := StringToDatetime(t.Format(format) + ":59:59")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func WeekStart(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> 7
	}

	startOfWeek := time.Date(
		t.Year(),
		t.Month(),
		t.Day()-weekday+1,
		0, 0, 0, 0,
		t.Location(),
	)
	result := StringToDatetime(startOfWeek)

	if result != nil {
		return *result
	}

	return t.UTC()
}

func WeekEnd(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	// Sunday = last day of ISO week
	lastDayOfWeek := time.Date(
		t.Year(),
		t.Month(),
		t.Day()+(7-weekday),
		0, 0, 0, 0,
		t.Location(),
	)
	result := StringToDatetime(lastDayOfWeek.Format(time.DateOnly) + " 23:59:59")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func MonthStart(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	format := "2006-01"
	result := StringToDatetime(t.Format(format) + "-01 00:00:00")

	if result != nil {
		return *result
	}

	return t.UTC()
}

func MonthEnd(value interface{}) time.Time {
	t := StringToDatetime(value)
	if t == nil {
		_t := time.Now()
		t = &_t
	}

	_t := time.Date(
		t.Year(),
		t.Month()+1,
		0, // day 0 = last day of previous month
		0, 0, 0, 0,
		t.Location(),
	)
	t = &_t

	result := StringToDatetime(t)

	if result != nil {
		return *result
	}

	return t.UTC()
}

func ObjectID(id interface{}) bson.ObjectID {
	var value bson.ObjectID

	// console.Warn("id.before", id)

	if v, ok := id.(string); ok {
		value, _ = bson.ObjectIDFromHex(v)
	} else if v, ok := id.(int); ok {
		value, _ = bson.ObjectIDFromHex(fmt.Sprint(v))
	} else if v, ok := id.(bson.ObjectID); ok {
		value = v
	} else {
		value = bson.NewObjectID()
	}

	// console.Warn("id.after", value)

	return value
}

func ObjectIDtoString(id bson.ObjectID) string {
	value := id.Hex()

	return value
}

func StringLength(value string) int {
	return utf8.RuneCountInString(value)
}

func GetChildRelativeName(parent string, collection string, primaryKey string, foreignKey string) string {
	var classVariable = ToVariable(collection)
	var testKey = ToUnderscore(foreignKey)

	if strings.HasSuffix(testKey, "_id") {
		newVariable := ToVariable(testKey[0:(StringLength(testKey) - 3)])
		classVariable = ToVariable(collection)

		if ToVariable(parent) != ToVariable(newVariable) {
			classVariable = ToVariable(newVariable + "_" + collection)
		}
	} else {
		classVariable = ToVariable(testKey + "_" + collection)
	}

	classVariable = Pluralize(classVariable)

	return classVariable
}

func GetParentRelativeName(collection string, primaryKey string, foreignKey string) string {

	var classVariable = ToVariable(foreignKey)
	var testKey = ToUnderscore(foreignKey)

	if strings.HasSuffix(testKey, "_id") {
		classVariable = ToVariable(testKey[0:(StringLength(testKey) - 3)])
	}

	if classVariable == foreignKey || classVariable == Singularize(foreignKey) {
		classVariable = ToVariable(ToUnderscore(foreignKey) + "_info")
	}

	return classVariable
}

func TrackTime(start *time.Time, name string) {
	elapsed := time.Since(*start)
	fmt.Printf("%s took %s\n", name, elapsed)
	*start = time.Now()
}

func HomeDirectory(name string) string {
	dir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("HomeDirectory", err.Error())
	}

	if dir == "/" || IsEmpty(dir) {
		dir = "/root"
	}

	appDir := dir + string(os.PathSeparator) + ".yekonga-server" + string(os.PathSeparator) + name

	if info, err := os.Stat(appDir); err != nil {
		if info != nil && !info.IsDir() {
			os.MkdirAll(appDir, 0755)
		} else {
			err := os.MkdirAll(appDir, 0755)
			if err != nil {
				console.Error("HomeDirectory", appDir, err)
			}
		}
	}

	return appDir
}

func GetBaseUrl(str string) string {
	ip, _ := GetLocalIP()
	port := config.Config.Ports.Server

	if config.Config.Ports.Secure {
		port = config.Config.Ports.SSLServer
	}

	if regexp.MustCompile(`http(s?)://`).MatchString(str) {
		return str
	}

	return "http://" + ip + ":" + strconv.Itoa(port) + "/" + str
}

// GetDirectoryPath returns the absolute path of the specified file or executable
func GetDirectoryPath() string {
	// Get the absolute path of the executable
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve the absolute path
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return ""
	}
	return absPath
}

// GetPublicPath returns the absolute path of a frontend file (e.g., index.html)
func GetPublicPath() (string, error) {
	// Assuming the frontend files are in the 'frontend/dist' directory
	// Adjust the path based on your project structure
	frontendPath := filepath.Join("public")
	absPath, err := filepath.Abs(frontendPath)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func GetPath(relativePath string) string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}

	// 2. Get the directory of the executable
	exPath := filepath.Dir(ex)

	// 3. Join the executable's directory with the relative path
	absolutePath := filepath.Join(exPath, relativePath)

	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	return absolutePath
}

func GetValueOfString(data interface{}, key string) string {
	return GetMapString(data, key)
}

func GetMapString(data interface{}, key string) string {
	if value, ok := GetMapValue(data, key).(string); ok {
		return value
	} else if value, ok := GetMapValue(data, key).(bson.ObjectID); ok {
		return ObjectIDtoString(value)
	}

	return ""
}

func GetValueOfInt(data interface{}, key string) int {
	return GetMapInt(data, key)
}

func GetMapInt(data interface{}, key string) int {
	if value, ok := GetMapValue(data, key).(int); ok {
		return value
	}

	return 0
}

func GetValueOfFloat(data interface{}, key string) int {
	return GetMapInt(data, key)
}

func GetMapFloat(data interface{}, key string) float64 {
	if value, ok := GetMapValue(data, key).(int); ok {
		return ToFloat(value)
	} else if value, ok := GetMapValue(data, key).(float64); ok {
		return value
	}

	return 0
}

func GetValueOfBoolean(data interface{}, key string) bool {
	return GetMapBoolean(data, key)

}

func GetMapBoolean(data interface{}, key string) bool {
	if value, ok := GetMapValue(data, key).(bool); ok {
		return value
	}

	return false
}

func GetValueOfDate(data interface{}, key string) time.Time {
	return GetMapDate(data, key)
}

func GetMapDate(data interface{}, key string) time.Time {
	value := GetMapValue(data, key)
	// console.Log("value", GetType(value), value)

	return GetTimestamp(value)
}

func GetValueOfMap(data interface{}, key string) map[string]interface{} {
	return GetMap(data, key)
}

func GetMap(data interface{}, key string) map[string]interface{} {
	if value, ok := GetMapValue(data, key).(map[string]interface{}); ok {
		return value
	}

	return nil
}

func GetValueOf(data interface{}, key string) interface{} {
	return GetMapValue(data, key)
}

// func getMapValueItem() {
// 	if vi, oki := v[first]; oki {
// 		if len(keys[1:]) == 0 {
// 			return vi
// 		} else {
// 			last := strings.Join(keys[1:], ".")
// 			return GetMapValue(vi, last)
// 		}
// 	}
// }

func GetMapValue(data interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	first := keys[0]
	var localData interface{}

	if IsNotEmpty(data) {
		if IsPointer(data) {
			v := reflect.ValueOf(data).Elem()
			if v.IsValid() {
				localData = v.Interface()
			}
		} else {
			localData = data
		}
	}

	if IsMap(localData) {

		if v, ok := localData.(map[string]interface{}); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		} else if v, ok := localData.(datatype.DataMap); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		} else if v, ok := localData.(datatype.Context); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		} else if v, ok := localData.(datatype.ContextObject); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		} else if v, ok := localData.(datatype.JsonObject); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		} else if v, ok := localData.(bson.M); ok {
			if vi, oki := v[first]; oki {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		}
	} else if v, ok := localData.([]interface{}); ok {
		if IsNumeric(first) {
			pos := ToInt(first)

			if vi := v[pos]; vi != nil {
				if len(keys[1:]) == 0 {
					return vi
				} else {
					last := strings.Join(keys[1:], ".")
					return GetMapValue(vi, last)
				}
			}
		}
	}

	return nil
}

func GetMapArray(data interface{}, source string, keys map[string]string) []interface{} {
	list := []interface{}{}
	dataList := GetMapValue(data, source)

	if v, ok := dataList.([]interface{}); ok {
		for _, vi := range v {
			data := map[string]interface{}{}
			for kii, vii := range keys {
				data[kii] = GetMapValue(vi, vii)
			}

			list = append(list, data)
		}
	}

	return list
}

func GetFirst(data interface{}) interface{} {
	if value, ok := data.(map[string]interface{}); ok {
		for _, v := range value {
			return v
		}
	} else if value, ok := data.([]interface{}); ok {
		for _, v := range value {
			return v
		}
	} else if value, ok := data.([]map[string]interface{}); ok {
		for _, v := range value {
			return v
		}
	}

	return nil
}

func GetType(data interface{}) string {
	return fmt.Sprintf("%T", data)
}

func IsSliceOfMapStringInterface(v interface{}) bool {
	t := reflect.TypeOf(v)
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Map &&
		t.Elem().Key().Kind() == reflect.String && t.Elem().Elem().Kind() == reflect.Interface
}

func GetList(data interface{}, key string) []interface{} {
	var source interface{}
	list := []interface{}{}

	if IsPointer(data) {
		source = reflect.ValueOf(data).Elem().Interface()
	} else {
		source = data
	}

	if source != nil {
		// Use reflection to check if source is a slice
		converted := ToMapList[interface{}](source)

		for _, vi := range converted {
			if vii, okii := vi[key]; okii {
				list = append(list, vii)
			}
		}
	}

	if len(list) == 0 {
		logger.Error("GetList result", list)
	}

	return list
}

func IsPointer(v interface{}) bool {
	if v == nil {
		return false
	}

	return reflect.TypeOf(v).Kind() == reflect.Ptr
}

func CreateFile(data interface{}, filename string) error {
	return SaveToFile(data, filename)
}

func SaveToFile(data interface{}, filename string) error {
	var (
		rowData []byte
		err     error
	)
	// Convert to JSON
	if d, ok := data.(string); ok {
		rowData = []byte(d)
	} else {
		rowData, err = json.MarshalIndent(data, "", "  ") // Pretty print with indentation
		if err != nil {
			return err
		}
	}

	// Extract directory path
	dir := filepath.Dir(filename)

	// Create all folders if they don't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(filename, rowData, 0644) // 0644 is standard file permission
	if err != nil {
		return err
	}

	return nil
}

func CreateDirectory(dir string) error {
	// Create all folders if they don't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return nil
}

func CreateFolder(dir string) error {
	return CreateDirectory(dir)
}

func ExtractGraphqlQuery(data interface{}, level uint) map[uint][]string {
	var result map[uint][]string = map[uint][]string{}
	if d, ok := data.(map[string]interface{}); ok {
		for k := range d {
			var list interface{}
			if k == "Definitions" {
				list = GetMapValue(data, "Definitions")
			} else if k == "SelectionSet" {
				list = GetMapValue(data, "SelectionSet.Selections")
			} else {

			}
			if result[level] == nil {
				result[level] = []string{}
			}

			if l, ok := list.([]interface{}); ok {
				for _, vi := range l {
					newLevel := level + 1
					vn := GetMapValue(vi, "Name.Value")
					if vn == nil {
						newLevel = level
					}

					rs := ExtractGraphqlQuery(vi, newLevel)

					if vii, ok := vn.(string); ok {
						if len(rs[level+1]) > 0 {
							result[level] = append(result[level], "_c_"+vii)
							result[level+1] = append(result[level+1], "_p_"+vii)
						} else {
							result[level] = append(result[level], vii)
						}
					}

					if result[level+1] == nil {
						result[level+1] = []string{}
					}

					for k, v := range rs {
						if len(v) > 0 {
							result[k] = append(result[k], v...)
						}
					}
				}

			}
		}
	} else if d, ok := data.([]interface{}); ok {
		count := len(d)
		for i := 0; i < count; i++ {
			v := d[i]
			logger.Error("3", v)
		}
	}

	return result
}

func Get(url string, headers map[string]string) (status int, responseBody string, err error) {
	return GetRequest(url, headers)
}

// GetRequest performs an HTTP GET request with the specified URL, headers, and optional body.
func GetRequest(url string, headers map[string]string) (status int, responseBody string, err error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create the request with optional body
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return 0, "", fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Make the GET request
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("error reading response: %v", err)
	}

	return resp.StatusCode, string(respBody), nil
}

func Post(url string, headers map[string]string, body interface{}) (status int, responseBody string, err error) {
	return PostRequest(url, headers, body)
}

func PostRequest(url string, headers map[string]string, body interface{}) (status int, responseBody string, err error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Serialize body to JSON
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, "", fmt.Errorf("error marshaling body: %v", err)
	}

	// Create the request with the serialized body
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Make the POST request
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("error reading response: %v", err)
	}

	return resp.StatusCode, string(respBody), nil
}

func ValidateEmail(email interface{}) bool {
	if v, ok := email.(string); ok {
		re := regexp.MustCompile(`^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`)
		return re.MatchString(strings.ToLower(v))
	}

	return false
}

func IsEmail(email interface{}) bool {
	return ValidateEmail(email)
}

func IsPhone(value interface{}) bool {
	if v, ok := value.(string); ok {
		if v == "" {
			return false
		}
		v = FormatPhone(v) // Assuming formatPhone is defined elsewhere

		phone := regexp.MustCompile(`(?im)^[\+]?[(]?[0-9]{3}[)]?[-\s\.]?[0-9]{3}[-\s\.]?[0-9]{4,6}$`)
		return phone.MatchString(v)

	}

	return false
}

// modifyString is a placeholder implementation; adjust based on your needs
func ModifyString(value string) string {
	// Example: Randomly shuffle characters (if that's what modifyString does)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	runes := []rune(value)
	r.Shuffle(len(runes), func(i, j int) {
		runes[i], runes[j] = runes[j], runes[i]
	})
	return string(runes)
}

func FormatPhone(phone interface{}) string {
	if value, ok := phone.(string); ok && IsNotEmpty(value) {
		value = strings.ReplaceAll(value, " ", "")
		value = strings.ReplaceAll(value, "-", "")
		value = strings.ReplaceAll(value, "_", "")
		value = strings.ReplaceAll(value, ".", "")
		value = strings.ReplaceAll(value, ",", "")

		if strings.HasPrefix(value, "+") {
			value = value[1:]
		} else if strings.HasPrefix(value, "255") {
			// No change needed
		} else if strings.HasPrefix(value, "0") && len(value) == 10 {
			value = "255" + value[1:]
		} else if !strings.HasPrefix(value, "0") && len(value) == 9 {
			value = "255" + value
		}

		if len(value) < 10 || (strings.HasPrefix(value, "255") && len(value) != 12) {
			return ""
		}

		return value
	}

	return ""
}

func PhoneFormat(phone interface{}) string {
	return FormatPhone(phone)
}

func GetRandomString(length int, mode string) string {
	var chars string
	n := "0123456789" + fmt.Sprintf("%d", time.Now().UnixMilli())
	l := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	h := "ABCDEFabcdef"

	switch mode {
	case "number":
		chars = n
	case "letter":
		chars = l
	case "hex":
		chars = n + h
	default:
		chars = n + l
	}

	result := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < length; i++ {
		// JavaScript logic: for 'number' type, avoid leading zero
		if mode == "number" && i == 0 && len(chars) > 1 {
			// Select from chars[1:] to skip '0'
			result[i] = chars[1+r.Intn(len(chars)-1)]
		} else {
			result[i] = chars[r.Intn(len(chars))]
		}
	}

	return string(result)
}

func GetRandomInt(length int) string {
	return GetRandomString(length, "number")
}

func GetHexString(length int) string {
	return GetRandomString(length, "hex")
}

func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetNestedValue returns the value at the dot-separated path or nil if not found
func GetNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if current == nil {
			return nil
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// TextTemplate replaces {{key}} and {{nested.key}} placeholders
// Uses the safer FindAllStringSubmatchIndex approach to avoid infinite loops
func TextTemplate(templateString string, data map[string]interface{}, customPattern *regexp.Regexp) string {
	if customPattern == nil {
		// Supports: {{ name }}, {{ user.name }}, {{ items.0.price }}, spaces around name
		customPattern = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)*)\s*}}`)
	}

	result := []byte(templateString)
	var buf strings.Builder

	lastIndex := 0

	for _, match := range customPattern.FindAllStringSubmatchIndex(string(result), -1) {
		// match[0] = start of whole match
		// match[1] = start of capture group (the key)
		// match[2] = end   of capture group

		keyStart, keyEnd := match[2], match[3]
		key := string(result[keyStart:keyEnd])

		// Write text before this placeholder
		buf.Write(result[lastIndex:match[0]])

		// Get value and convert to string
		value := GetNestedValue(data, key)
		if value != nil {
			buf.WriteString(fmt.Sprintf("%v", value))
		} // else → leave empty (you can also write "MISSING" or similar)

		lastIndex = match[1] // end of this match
	}

	// Write remaining text after last match
	buf.Write(result[lastIndex:])

	return buf.String()
}

// getTextContent reads a template file and processes it with data
func GetTextContent(template string, data map[string]interface{}) string {
	dirname := GetDirectoryPath()
	content := ""
	templatePath := ""
	// Check if file exists in Dirname
	if _, err := os.Stat(filepath.Join(dirname, template)); err == nil {
		templatePath = filepath.Join(dirname, template)
	} else if _, err := os.Stat(filepath.Join(dirname, template)); err == nil {
		// Check if file exists in Src
		templatePath = filepath.Join(dirname, template)
	}

	if templatePath != "" {
		text, err := os.ReadFile(templatePath)
		if err != nil {
			// Log error (replace with proper logging if needed)
			println("Error reading file:", err.Error())
			return content
		}
		content = TextTemplate(string(text), data, nil)
		// Assuming clearSpecialCharacters is defined elsewhere
		content = ClearSpecialCharacters(content)
	}

	return content
}

// clearSpecialCharacters cleans the input string by replacing curly apostrophes,
// removing HTML tags, and keeping only allowed characters.
func ClearSpecialCharacters(val string) string {
	// If val is empty, return an empty string
	if val == "" {
		return ""
	}

	// Replace curly apostrophes (’) with straight apostrophes (')
	val = strings.ReplaceAll(val, "’", "'")

	// Remove HTML tags (e.g., <p>, <div>, etc.)
	reHTML := regexp.MustCompile(`<[^>]+>`)
	val = reHTML.ReplaceAllString(val, "")

	// Keep only allowed characters: a-z, A-Z, 0-9, '"?!.,;:-_&()+\s
	reAllowed := regexp.MustCompile(`[^a-zA-Z0-9'"?!.,;:-_&()+\s]`)
	val = reAllowed.ReplaceAllString(val, "")

	return val
}

// getWhatsappContent processes template content and attempts to parse it as JSON
func GetWhatsappContent(template string, data map[string]interface{}) interface{} {
	content := GetTextContent(template, data)

	var result interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Ignore JSON parse error, return content as-is
		return content
	}

	return result
}

// getEmailContent reads a template file, processes it, and wraps it in a layout
func GetEmailContent(layout, template string, data map[string]interface{}) string {
	dirname := GetDirectoryPath()
	content := ""
	templatePath := ""
	// Check if file exists in Dirname
	if _, err := os.Stat(filepath.Join(dirname, template)); err == nil {
		templatePath = filepath.Join(dirname, template)
	} else if _, err := os.Stat(filepath.Join(dirname, template)); err == nil {
		// Check if file exists in Src
		templatePath = filepath.Join(dirname, template)
	}

	if templatePath != "" {
		html, err := os.ReadFile(templatePath)
		if err != nil {
			// Log error (replace with proper logging if needed)
			println("Error reading file:", err.Error())
			return content
		}
		content = TextTemplate(string(html), data, nil)
	}

	// Assuming getEmailLayout is defined elsewhere
	return getEmailLayout(layout, content, data)
}

// getEmailLayout processes an email layout template with content and server.Configuration data
func getEmailLayout(layout, content string, data map[string]interface{}) string {
	dirname := GetDirectoryPath()
	temp := ""
	website := GetBaseUrl("") // Assuming getBaseUrl is defined elsewhere
	logo := website + "/img/mail/@notificationId/logo.png"
	appName := config.Config.AppName
	year := ToTimestampString(nil, "2006") // Go uses "2006" for YYYY
	primaryColor := "#306da7"
	secondaryColor := "#306da7"
	darkBackgroundColor := "#033360"

	if IsNotEmpty(config.Config.Branding) {
		if IsNotEmpty(config.Config.Branding.PrimaryColor) {
			primaryColor = config.Config.Branding.PrimaryColor
		}

		if IsNotEmpty(config.Config.Branding.SecondaryColor) {
			secondaryColor = config.Config.Branding.SecondaryColor
		}

		if IsNotEmpty(config.Config.Branding.DarkBackgroundColor) {
			darkBackgroundColor = config.Config.Branding.DarkBackgroundColor
		}
	}

	if layout != "" {
		filePath := filepath.Join(dirname, layout)
		if data, err := os.ReadFile(filePath); err == nil {
			temp = string(data)
		} else {
			// Log error (replace with proper logging if needed)
			println("Error reading layout file:", err.Error())
		}
	}

	if temp == "" || strings.TrimSpace(temp) == "" {
		if config.Config.EmailTemplate != "" {
			filePath := filepath.Join(dirname, config.Config.EmailTemplate)
			if data, err := os.ReadFile(filePath); err == nil {
				temp = string(data)
			} else {
				// Log error (replace with proper logging if needed)
				println("Error reading email template file:", err.Error())
			}
		}
	}

	if temp == "" || strings.TrimSpace(temp) == "" {
		filePath := filepath.Join(dirname, "assets/mail/email.html")
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Log error (replace with proper logging if needed)
			println("Error reading default email template:", err.Error())
			return ""
		}
		temp = string(data)
	}

	htmlContent := TextTemplate(temp, map[string]interface{}{
		"content":             content,
		"logo":                logo,
		"website":             website,
		"appName":             appName,
		"year":                year,
		"primaryColor":        primaryColor,
		"secondaryColor":      secondaryColor,
		"darkBackgroundColor": darkBackgroundColor,
	}, nil)

	return htmlContent
}

// ExtractDomain extracts the domain from a URL string
func ExtractDomain(input string) string {
	// Add scheme if missing
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		input = "https://" + input
	}

	// Parse the URL
	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	return u.Host
}
