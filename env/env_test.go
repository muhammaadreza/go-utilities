package env

import (
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testKey = "ENV_UTIL_TEST_VALUE"

func unsetForTest(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "temporary")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

func assertError(t *testing.T, err, want error, key, expected string, index int) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v; want errors.Is(_, %v)", err, want)
	}
	var detail *Error
	if !errors.As(err, &detail) {
		t.Fatalf("error %T does not contain *Error", err)
	}
	if detail.Expected() != expected {
		t.Fatalf("Expected() = %q; want %q", detail.Expected(), expected)
	}

	indexed, hasIndexMethod := any(detail).(interface {
		Index() (int, bool)
	})
	keyMatches := detail.Key() == key
	if !hasIndexMethod && index >= 0 {
		keyMatches = keyMatches || detail.Key() == fmt.Sprintf("%s[%d]", key, index)
	}
	if !keyMatches {
		t.Fatalf("Key() = %q; want %q (element index %d)", detail.Key(), key, index)
	}
	if hasIndexMethod {
		gotIndex, ok := indexed.Index()
		if index < 0 {
			if ok {
				t.Fatalf("Index() = (%d, true); want no element index", gotIndex)
			}
		} else if !ok || gotIndex != index {
			t.Fatalf("Index() = (%d, %v); want (%d, true)", gotIndex, ok, index)
		}
	}
}

func checkParse[T types](t *testing.T, raw string, want T, wantErr error) {
	t.Helper()
	t.Run(fmt.Sprintf("%T/%q", want, raw), func(t *testing.T) {
		got, err := parse[T](raw)
		if !reflect.DeepEqual(got, want) || !errors.Is(err, wantErr) {
			t.Fatalf("parse(%q) = (%v, %v); want (%v, %v)", raw, got, err, want, wantErr)
		}
	})
}

func TestParseScalars(t *testing.T) {
	checkParse(t, "  secret\t", "  secret\t", nil)
	checkParse(t, "", "", nil)
	checkParse(t, "   ", "   ", nil)
	checkParse(t, "\u2003 42\t", int(42), nil)
	checkParse(t, "+42", int(42), nil)
	checkParse(t, "010", int(10), nil)
	checkParse(t, "127", int8(127), nil)
	checkParse(t, "-128", int8(-128), nil)
	checkParse(t, "128", int8(0), ErrInvalidRange)
	checkParse(t, "-129", int8(0), ErrInvalidRange)
	checkParse(t, "32767", int16(32767), nil)
	checkParse(t, "-32768", int16(-32768), nil)
	checkParse(t, "32768", int16(0), ErrInvalidRange)
	checkParse(t, "-32769", int16(0), ErrInvalidRange)
	checkParse(t, "2147483647", int32(math.MaxInt32), nil)
	checkParse(t, "-2147483648", int32(math.MinInt32), nil)
	checkParse(t, "2147483648", int32(0), ErrInvalidRange)
	checkParse(t, "-2147483649", int32(0), ErrInvalidRange)
	checkParse(t, "9223372036854775807", int64(math.MaxInt64), nil)
	checkParse(t, "-9223372036854775808", int64(math.MinInt64), nil)
	checkParse(t, "9223372036854775808", int64(0), ErrInvalidRange)
	checkParse(t, "-9223372036854775809", int64(0), ErrInvalidRange)
	checkParse(t, "255", uint8(255), nil)
	checkParse(t, "256", uint8(0), ErrInvalidRange)
	checkParse(t, "65535", uint16(65535), nil)
	checkParse(t, "65536", uint16(0), ErrInvalidRange)
	checkParse(t, "4294967295", uint32(math.MaxUint32), nil)
	checkParse(t, "4294967296", uint32(0), ErrInvalidRange)
	checkParse(t, "18446744073709551615", uint64(math.MaxUint64), nil)
	checkParse(t, "18446744073709551616", uint64(0), ErrInvalidRange)
	checkParse(t, "-1", uint(0), ErrInvalidValue)
	checkParse(t, "+1", uint(0), ErrInvalidValue)
	checkParse(t, "0", uint(0), nil)
	checkParse(t, "1.5", float32(1.5), nil)
	checkParse(t, "1e39", float32(0), ErrInvalidRange)
	checkParse(t, "1.5", float64(1.5), nil)
	checkParse(t, "1e309", float64(0), ErrInvalidRange)
	checkParse(t, "1e-1000", float64(0), nil)
	checkParse(t, " 1e2 ", float64(100), nil)
	checkParse(t, " 1m30s ", 90*time.Second, nil)
	checkParse(t, "-5s", -5*time.Second, nil)
	checkParse(t, "0", time.Duration(0), nil)
	checkParse(t, "1.5ms", 1500*time.Microsecond, nil)
	checkParse(t, "9223372036854775807ns", time.Duration(math.MaxInt64), nil)
	checkParse(t, "-9223372036854775808ns", time.Duration(math.MinInt64), nil)
	checkParse(t, "9223372036854775808ns", time.Duration(0), ErrInvalidValue)
	checkParse(t, "-9223372036854775809ns", time.Duration(0), ErrInvalidValue)
	checkParse(t, "5", time.Duration(0), ErrInvalidValue)
	checkParse(t, "1d", time.Duration(0), ErrInvalidValue)
	checkParse(t, "2026-10-18T10:57:22.123Z", time.Date(2026, 10, 18, 10, 57, 22, 123000000, time.UTC), nil)
	checkParse(t, "2026-10-18", time.Time{}, ErrInvalidValue)
	checkParse(t, "2026-02-30T00:00:00Z", time.Time{}, ErrInvalidValue)
	checkParse(t, "\u2003\t", time.Time{}, ErrEmptyValue)
	checkParse(t, "\n", time.Duration(0), ErrEmptyValue)
	checkParse(t, " ", false, ErrEmptyValue)
	checkParse(t, " ", float32(0), ErrEmptyValue)
	checkParse(t, " ", uint16(0), ErrEmptyValue)

	for _, raw := range []string{"abc", "4 2", "0x10", "0b10", "1_000", "１２"} {
		checkParse(t, raw, int(0), ErrInvalidValue)
	}
	for _, raw := range []string{"", "  ", "\u00a0\t\n"} {
		checkParse(t, raw, int(0), ErrEmptyValue)
	}
	for _, raw := range []string{"true", "TRUE", "True", "t", "T", "1", " true "} {
		checkParse(t, raw, true, nil)
	}
	for _, raw := range []string{"false", "FALSE", "False", "f", "F", "0"} {
		checkParse(t, raw, false, nil)
	}
	for _, raw := range []string{"yes", "no", "TrUe", "2"} {
		checkParse(t, raw, false, ErrInvalidValue)
	}
	for _, raw := range []string{"NaN", "nan", "Inf", "+Inf", "-Inf", "Infinity", "-INFINITY", " NaN "} {
		checkParse(t, raw, float32(0), ErrInvalidValue)
		checkParse(t, raw, float64(0), ErrInvalidValue)
	}
	checkParse(t, strconv.FormatUint(uint64(^uint(0)), 10), ^uint(0), nil)
	checkParse(t, strconv.Itoa(int(^uint(0)>>1)), int(^uint(0)>>1), nil)
	checkParse(t, "18446744073709551616", uint(0), ErrInvalidRange)
	checkParse(t, "9223372036854775808", int(0), ErrInvalidRange)
	if strconv.IntSize == 32 {
		checkParse(t, "4294967296", uint(0), ErrInvalidRange)
		checkParse(t, "2147483648", int(0), ErrInvalidRange)
	}
}

func TestTimeZonePreserved(t *testing.T) {
	t.Setenv(testKey, " 2026-10-18T18:00:00+07:00 ")
	got, err := Required[time.Time](testKey)
	want := time.Date(2026, 10, 18, 11, 0, 0, 0, time.UTC)
	_, offset := got.Zone()
	if err != nil || !got.Equal(want) || offset != 7*60*60 {
		t.Fatalf("got %v, %v; want equivalent instant %v with +07:00 offset", got, err, want)
	}
}

func TestScalarGetters(t *testing.T) {
	for _, tc := range []struct {
		name        string
		present     bool
		raw         string
		required    int
		requiredErr error
		optional    int
		optionalErr error
	}{
		{name: "missing", requiredErr: ErrMissingValue, optional: 9},
		{name: "empty", present: true, requiredErr: ErrEmptyValue, optional: 9},
		{name: "whitespace", present: true, raw: " \t", requiredErr: ErrEmptyValue, optionalErr: ErrEmptyValue},
		{name: "invalid", present: true, raw: "abc", requiredErr: ErrInvalidValue, optionalErr: ErrInvalidValue},
		{name: "overflow", present: true, raw: "9223372036854775808", requiredErr: ErrInvalidRange, optionalErr: ErrInvalidRange},
		{name: "zero", present: true, raw: "0"},
		{name: "positive", present: true, raw: "42", required: 42, optional: 42},
		{name: "trimmed", present: true, raw: " 42\t", required: 42, optional: 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.present {
				t.Setenv(testKey, tc.raw)
			} else {
				unsetForTest(t, testKey)
			}
			got, err := Required[int](testKey)
			if got != tc.required {
				t.Fatalf("Required = %d; want %d", got, tc.required)
			}
			if tc.requiredErr != nil {
				assertError(t, err, tc.requiredErr, testKey, "int", -1)
			} else if err != nil {
				t.Fatal(err)
			}
			got, err = Optional(testKey, 9)
			if got != tc.optional {
				t.Fatalf("Optional = %d; want %d", got, tc.optional)
			}
			if tc.optionalErr != nil {
				assertError(t, err, tc.optionalErr, testKey, "int", -1)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestScalarStringsAndBool(t *testing.T) {
	for _, raw := range []string{" ", " secret ", "line one\nline two"} {
		t.Setenv(testKey, raw)
		got, err := Required[string](testKey)
		if err != nil || got != raw {
			t.Fatalf("Required = %q, %v; want exact string %q", got, err, raw)
		}
		got, err = Optional(testKey, "fallback")
		if err != nil || got != raw {
			t.Fatalf("Optional = %q, %v; want exact string %q", got, err, raw)
		}
	}
	t.Setenv(testKey, "false")
	if got, err := Optional(testKey, true); got || err != nil {
		t.Fatalf("false was replaced with fallback: %v, %v", got, err)
	}
	t.Setenv(testKey, "")
	if got, err := Optional(testKey, "fallback"); got != "fallback" || err != nil {
		t.Fatal(got, err)
	}
}

func TestAliases(t *testing.T) {
	type portAlias = uint16
	t.Setenv(testKey, "65")
	if got, err := Required[portAlias](testKey); got != 65 || err != nil {
		t.Fatal(got, err)
	}
	if got, err := Required[byte](testKey); got != 65 || err != nil {
		t.Fatal(got, err)
	}
	if got, err := Required[rune](testKey); got != 65 || err != nil {
		t.Fatal(got, err)
	}
}

func TestInvalidKeys(t *testing.T) {
	for _, key := range []string{"", "A=B", "A\x00B"} {
		t.Run(strconv.Quote(key), func(t *testing.T) {
			got, err := Required[int](key)
			assertError(t, err, ErrInvalidKey, key, "int", -1)
			if got != 0 {
				t.Fatal("Required returned a nonzero value")
			}
			got, err = Optional(key, 42)
			assertError(t, err, ErrInvalidKey, key, "int", -1)
			if got != 0 {
				t.Fatal("Optional used fallback for invalid key")
			}
			items, err := RequiredSlice[int](key, ",")
			assertError(t, err, ErrInvalidKey, key, "[]int", -1)
			if items != nil {
				t.Fatal("RequiredSlice returned a result")
			}
			items, err = OptionalSlice(key, ",", []int{42})
			assertError(t, err, ErrInvalidKey, key, "[]int", -1)
			if items != nil {
				t.Fatal("OptionalSlice used fallback for invalid key")
			}
		})
	}
}

func TestLiteralKey(t *testing.T) {
	t.Setenv(" ENV_UTIL_LITERAL_KEY ", "42")
	got, err := Required[int](" ENV_UTIL_LITERAL_KEY ")
	if got != 42 || err != nil {
		t.Fatal("key was normalized", got, err)
	}
}

func TestRequiredSlice(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      string
		sep      string
		want     []int
		wantErr  error
		index    int
		expected string
	}{
		{name: "single", raw: "42", sep: ",", want: []int{42}},
		{name: "trimmed", raw: " 1, 2\t,3 ", sep: ",", want: []int{1, 2, 3}},
		{name: "multiple bytes", raw: "1::2::3", sep: "::", want: []int{1, 2, 3}},
		{name: "unicode separator", raw: "1🔥2", sep: "🔥", want: []int{1, 2}},
		{name: "digit separator", raw: "212", sep: "1", want: []int{2, 2}},
		{name: "literal space separator", raw: "1 2", sep: " ", want: []int{1, 2}},
		{name: "empty value", sep: ",", wantErr: ErrEmptyValue, index: -1, expected: "[]int"},
		{name: "empty separator", raw: "1,2", wantErr: ErrEmptySeparator, index: -1, expected: "[]int"},
		{name: "leading empty", raw: ",1", sep: ",", wantErr: ErrEmptyValue, index: 0, expected: "int"},
		{name: "middle empty", raw: "1,,2", sep: ",", wantErr: ErrEmptyValue, index: 1, expected: "int"},
		{name: "trailing empty", raw: "1,2,", sep: ",", wantErr: ErrEmptyValue, index: 2, expected: "int"},
		{name: "blank element", raw: "1, \t,2", sep: ",", wantErr: ErrEmptyValue, index: 1, expected: "int"},
		{name: "blank value", raw: " \u2003", sep: ",", wantErr: ErrEmptyValue, index: 0, expected: "int"},
		{name: "invalid element", raw: "1,x,3", sep: ",", wantErr: ErrInvalidValue, index: 1, expected: "int"},
		{name: "overflow", raw: "1,9223372036854775808", sep: ",", wantErr: ErrInvalidRange, index: 1, expected: "int"},
		{name: "overlap", raw: "1aaa2", sep: "aa", wantErr: ErrInvalidValue, index: 1, expected: "int"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(testKey, tc.raw)
			got, err := RequiredSlice[int](testKey, tc.sep)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("slice = %v; want %v", got, tc.want)
			}
			if tc.wantErr != nil {
				assertError(t, err, tc.wantErr, testKey, tc.expected, tc.index)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
	unsetForTest(t, testKey)
	got, err := RequiredSlice[int](testKey, ",")
	assertError(t, err, ErrMissingValue, testKey, "[]int", -1)
	if got != nil {
		t.Fatal(got)
	}
}

func TestOtherSliceTypes(t *testing.T) {
	t.Setenv(testKey, " a , b ")
	if got, err := RequiredSlice[string](testKey, ","); err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatal(got, err)
	}
	t.Setenv(testKey, "true, 0, T")
	if got, err := RequiredSlice[bool](testKey, ","); err != nil || !reflect.DeepEqual(got, []bool{true, false, true}) {
		t.Fatal(got, err)
	}
	t.Setenv(testKey, "1s, 250ms, -1s")
	if got, err := RequiredSlice[time.Duration](testKey, ","); err != nil || !reflect.DeepEqual(got, []time.Duration{time.Second, 250 * time.Millisecond, -time.Second}) {
		t.Fatal(got, err)
	}
	t.Setenv(testKey, "2026-01-01T00:00:00Z|2026-01-02T00:00:00Z")
	if got, err := RequiredSlice[time.Time](testKey, "|"); err != nil || len(got) != 2 || got[1].Sub(got[0]) != 24*time.Hour {
		t.Fatal(got, err)
	}
	t.Setenv(testKey, "1.5,NaN")
	got, err := RequiredSlice[float64](testKey, ",")
	assertError(t, err, ErrInvalidValue, testKey, "float64", 1)
	if got != nil {
		t.Fatal("partial float result was returned")
	}
}

func TestOptionalSlice(t *testing.T) {
	for _, name := range []string{"missing", "empty"} {
		t.Run(name, func(t *testing.T) {
			if name == "missing" {
				unsetForTest(t, testKey)
			} else {
				t.Setenv(testKey, "")
			}
			fallback := []int{1, 2}
			got, err := OptionalSlice(testKey, ",", fallback)
			if err != nil || !reflect.DeepEqual(got, fallback) {
				t.Fatal(got, err)
			}
			got[0] = 99
			fallback[1] = 88
			if fallback[0] != 1 || got[1] != 2 {
				t.Fatal("fallback and result share their backing array")
			}
			for _, empty := range [][]int{nil, {}} {
				got, err := OptionalSlice(testKey, ",", empty)
				if err != nil || !reflect.DeepEqual(got, empty) {
					t.Fatalf("nilness changed: %#v, %v; want %#v", got, err, empty)
				}
			}
		})
	}

	t.Setenv(testKey, "0,2")
	got, err := OptionalSlice(testKey, ",", []int{9})
	if err != nil || !reflect.DeepEqual(got, []int{0, 2}) {
		t.Fatal(got, err)
	}
	for _, tc := range []struct {
		raw   string
		cause error
	}{
		{"1,x", ErrInvalidValue}, {"1, ", ErrEmptyValue},
	} {
		t.Setenv(testKey, tc.raw)
		got, err := OptionalSlice(testKey, ",", []int{9})
		assertError(t, err, tc.cause, testKey, "int", 1)
		if got != nil {
			t.Fatal("invalid value used the fallback")
		}
	}
	t.Setenv(testKey, "")
	got, err = OptionalSlice(testKey, "", []int{9})
	assertError(t, err, ErrEmptySeparator, testKey, "[]int", -1)
	if got != nil {
		t.Fatal("empty separator used the fallback")
	}
}

func TestFallbackIsTrusted(t *testing.T) {
	unsetForTest(t, testKey)
	if got, err := Optional(testKey, math.NaN()); err != nil || !math.IsNaN(got) {
		t.Fatal("typed fallback was validated", got, err)
	}
	if got, err := OptionalSlice(testKey, ",", []string{""}); err != nil || !reflect.DeepEqual(got, []string{""}) {
		t.Fatal("fallback element was validated", got, err)
	}
}

func TestErrorFormattingAndWrapping(t *testing.T) {
	t.Setenv(testKey, "1,not-a-real-secret")
	_, err := RequiredSlice[int](testKey, ",")
	wrapped := fmt.Errorf("load config: %w", err)
	assertError(t, wrapped, ErrInvalidValue, testKey, "int", 1)
	for cause := wrapped; cause != nil; cause = errors.Unwrap(cause) {
		if strings.Contains(cause.Error(), "not-a-real-secret") {
			t.Fatal("raw value retained in error chain")
		}
	}
	_, err = Required[int]("line\nbreak\x00")
	assertError(t, err, ErrInvalidKey, "line\nbreak\x00", "int", -1)
	if strings.ContainsAny(err.Error(), "\n\x00") {
		t.Fatal("key's control characters were not escaped")
	}
}

func FuzzSliceStrings(f *testing.F) {
	for _, seed := range [][2]string{
		{"a,b", ","}, {" a :: b ", "::"}, {"a,,b", ","},
		{"aaaa", "aa"}, {"a🔥b🔥", "🔥"}, {"\u2003a|b\t", "|"},
		{"", ","}, {"a\x00b", "\x00"},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, raw, separator string) {
		if separator == "" {
			t.Skip() // Public getters validate this before invoking the parser.
		}
		want := strings.Split(raw, separator)
		badIndex := -1
		for i := range want {
			want[i] = strings.TrimSpace(want[i])
			if want[i] == "" && badIndex < 0 {
				badIndex = i
			}
		}
		got, err := parseSlice[string](testKey, raw, separator)
		if badIndex >= 0 {
			assertError(t, err, ErrEmptyValue, testKey, "string", badIndex)
			if got != nil {
				t.Fatal("partial result was returned")
			}
		} else if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, %v; want %#v", got, err, want)
		}
	})
}

var benchmarkInts []int
var benchmarkInt int

func BenchmarkParseSliceInt(b *testing.B) {
	for b.Loop() {
		var err error
		benchmarkInts, err = parseSlice[int]("PORTS", "1,2,3,4,5,6,7,8,9,10", ",")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRequiredInt(b *testing.B) {
	b.Setenv(testKey, "5432")
	for b.Loop() {
		var err error
		benchmarkInt, err = Required[int](testKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}
