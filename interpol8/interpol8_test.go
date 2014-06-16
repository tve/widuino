package interpol8

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

// i8Test structure has the inputs and the expected output
type i8Test struct {
	in               []RawPoint
	start, end, step uint64
	maxFill          int
	out              []RawPoint
}

// Signature of function under test
type i8TestFunc func(t i8Test) []RawPoint

func PrintRaw(raw []RawPoint) string {
	str := "["
	for i, r := range raw {
		if i != 0 {
			str += ","
		}
		str += fmt.Sprintf("[%d,%f]", r.Asof, r.Value)
	}
	return str
}

func doI8Tests(t *testing.T, tests []i8Test, fun i8TestFunc) {
	for _, td := range tests {
		out := fun(td)
		if !reflect.DeepEqual(td.out, out) {
			t.Errorf("For %s", PrintRaw(td.in))
			t.Errorf("    %d..%d by %d mf=%d", td.start, td.end, td.step, td.maxFill)
			t.Errorf("Exp %s", PrintRaw(td.out))
			t.Errorf("Got %s", PrintRaw(out))
		}
	}
}

// Test lTrim

var ltTests = []i8Test{
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		15, 30, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		0, 30, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		25, 30, 10, 0,
		[]RawPoint{{20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		20, 30, 10, 0,
		[]RawPoint{{20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
}

func TestLTrim(t *testing.T) {
	doI8Tests(t, ltTests, func(td i8Test) []RawPoint {
		return lTrim(td.in, td.start, td.step)
	})
}

// Test rTrim

var rtTests = []i8Test{
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		15, 60, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		15, 35, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		15, 30, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}},
	},
	{
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}},
		15, 25, 10, 0,
		[]RawPoint{{10, 0}, {20, 0}, {30, 0}, {40, 0}},
	},
}

func TestRTrim(t *testing.T) {
	doI8Tests(t, rtTests, func(td i8Test) []RawPoint {
		return rTrim(td.in, td.end, td.step)
	})
}

// Test adjustEnd

var adjEndTests = []struct {
	start, end, step uint64
	count            int
	newEnd           uint64
	err              bool
}{
	{10, 100, 10, 9, 100, false},
	{10, 105, 10, 10, 110, false},
	{10, 9, 10, 0, 0, true},
	{10, 10, 10, 0, 0, true},
	{10, 11, 10, 1, 20, false},
	{10, 19, 10, 1, 20, false},
	{10, 20, 10, 1, 20, false},
	{10, 21, 10, 2, 30, false},
	{10, 10000000, 1, 0, 0, true},
}

func TestAdjustEnd(t *testing.T) {
	for _, td := range adjEndTests {
		c, e, err := adjustEnd(td.start, td.end, td.step)
		if c != td.count || e != td.newEnd || (err != nil) != td.err {
			t.Error("For", td, "got", c, e, err)
		}
	}

}

//===== Helpers for testing interpolation =====

const start = 10000
const count = 20
const step = 20

// Struct with raw input points and expected output interpolated points
type tt struct {
	in  []RawPoint
	out []IntPoint
}

// Fill an []IntPoints "expected" array with NaNs so we don't have to type so much
func resNaNFill(r []IntPoint) []IntPoint {
	if len(r) < count {
		extra := make([]IntPoint, count-len(r))
		for i := range extra {
			nan := math.NaN()
			extra[i] = IntPoint{start, nan, nan, nan}
		}
		// if what we have starts with {0,0,0} we prepend NaNs, else append
		if len(r) > 0 && r[0].Avg == 0 {
			nan := math.NaN()
			r[0] = IntPoint{start, nan, nan, nan}
			r = append(extra, r...)
		} else {
			r = append(r, extra...)
		}
	}
	// set all the timestamps
	for i := 1; i < len(r); i += 1 {
		r[i].Asof = r[0].Asof + uint64(i*step)
	}
	return r
}

// Format IntPoints so we can easily cut&paste into expected
// out: "[]IntPoints{ { 1, 1, 2 }, { 2, 2, 3 }, { 3, 3, 3 } } },"
func fmtInt(int []IntPoint) (str string) {
	str = fmt.Sprintf("[%d]IntPoint{ ", len(int))
	wasNaN := false
	for _, v := range int {
		if false && math.IsNaN(v.Avg) {
			if !wasNaN {
				str += "NaN... "
			}
			wasNaN = true
		} else {
			str += fmt.Sprint("{", v.Asof, ",", v.Avg, ",", v.Min, ",", v.Max, "}, ")
			wasNaN = false
		}
	}
	str += " }"
	return
}

func resEqual(a1, a2 []IntPoint) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i := range a1 {
		v1 := a1[i]
		v2 := a2[i]
		//fmt.Println("Comparing", v1, v2, "->", (math.Abs(v1.Avg-v2.Avg) > 0.001),
		//	math.IsNaN(v1.Avg), math.IsNaN(v2.Avg))
		if v1.Asof != v2.Asof {
			return false
		}
		if math.Abs(v1.Avg-v2.Avg) > 0.001 || math.IsNaN(v1.Avg) != math.IsNaN(v2.Avg) {
			return false
		}
		if math.Abs(v1.Min-v2.Min) > 0.001 || math.IsNaN(v1.Min) != math.IsNaN(v2.Min) {
			return false
		}
		if math.Abs(v1.Max-v2.Max) > 0.001 || math.IsNaN(v1.Max) != math.IsNaN(v2.Max) {
			return false
		}
	}
	return true
}

//===== Tests for interpolation of rates =====

// Test cases for interpolating rate data
const s = start
const e = start + count*step

var rateTests = []tt{
	{in: []RawPoint{{s + 0, 10}, {s + 20, 20}, {s + 30, 30}, {s + 50, 40}, {s + 100, 90}},
		out: []IntPoint{{s, 0.5, 0.5, 0.5}, {s, 0.75, 0.5, 1}, {s, 0.75, 0.5, 1}, {s, 1, 1, 1}, {s, 1, 1, 1}}},
	// tests using a single raw point
	{in: []RawPoint{{s - 10, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 0, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 10, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 20, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 30, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 1000, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 1010, 1}}, out: []IntPoint{}},
	// tests using two raw points
	{in: []RawPoint{{s - 10, 0}, {s + 0, 20}}, out: []IntPoint{}},
	{in: []RawPoint{{s - 10, 0}, {s + 10, 20}},
		out: []IntPoint{{s, 1, 1, 1}}},
	{in: []RawPoint{{s - 20, 0}, {s + 20, 20}},
		out: []IntPoint{{s, 0.5, 0.5, 0.5}}},
	{in: []RawPoint{{s - 20, 0}, {s + 30, 25}},
		out: []IntPoint{{s, 0.5, 0.5, 0.5}, {s, 0.5, 0.5, 0.5}}},
	{in: []RawPoint{{s - 20, 0}, {s + 40, 60}},
		out: []IntPoint{{s, 1, 1, 1}, {s, 1, 1, 1}}},
	{in: []RawPoint{{s - 20, 0}, {s + 100, 20}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 1000, 0}, {s + 1010, 20}}, out: []IntPoint{{}}},
	{in: []RawPoint{{e - 10, 0}, {e + 10, 20}},
		out: []IntPoint{{s, 0, 0, 0}, {s, 1, 1, 1}}},
	{in: []RawPoint{{e - 20, 0}, {e + 10, 60}},
		out: []IntPoint{{s, 0, 0, 0}, {s, 2, 2, 2}}},
	// tests using three raw points
	{in: []RawPoint{{s - 10, 0}, {s + 10, 20}, {s + 30, 60}},
		out: []IntPoint{{s, 1.5, 1, 2}, {s, 2, 2, 2}}},
}

func TestInterpolateRate(t *testing.T) {
	for i := range rateTests {
		// fill output with NaN and set times
		exp := resNaNFill(rateTests[i].out)
		// run the interpolation
		res, err := Raw(rateTests[i].in, Rate, start, start+count*step, step, 4*step)
		// shouldn't have gotten any error
		if err != nil {
			t.Fatal("Test", i, "unexpected error:", err)
		}
		// test we got the right result
		if !resEqual(res, exp) {
			t.Error("Test", i, "failed!\nInput:\n", rateTests[i].in,
				"\nOutput:\n", fmtInt(res), "\nExpected:\n", fmtInt(exp))
		}
	}
}

//===== Tests for interpolation of absolute data =====

// Test cases for interpolating absolute data
var absTests = []tt{
	{in: []RawPoint{{s + 0, 1}, {s + 20, 2}, {s + 30, 3}, {s + 50, 4}, {s + 100, 9}},
		out: []IntPoint{
			{s, 1.5, 1, 2}, {s, 2.875, 2, 3.5}, {s, 4.125, 3.5, 5},
			{s, 6, 5, 7}, {s, 8, 7, 9}, {s, 9, 9, 9}},
	},
	// tests using a single raw point
	{in: []RawPoint{{s + -10, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{s + 0, 1}},
		out: []IntPoint{{s, 1, 1, 1}}},
	{in: []RawPoint{{s + 10, 1}},
		out: []IntPoint{{s, 1, 1, 1}}},
	{in: []RawPoint{{s + 20, 1}},
		out: []IntPoint{{s, math.NaN(), math.NaN(), math.NaN()}, {s, 1, 1, 1}}},
	{in: []RawPoint{{s + 30, 1}},
		out: []IntPoint{{s, math.NaN(), math.NaN(), math.NaN()}, {s, 1, 1, 1}}},
	{in: []RawPoint{{e + 0, 1}}, out: []IntPoint{}},
	{in: []RawPoint{{e + 10, 1}}, out: []IntPoint{}},
	// tests using two raw points
	{in: []RawPoint{{s + -10, 1}, {s + 0, 2}},
		out: []IntPoint{{s, 2, 2, 2}}},
	{in: []RawPoint{{s + -10, 1}, {s + 10, 2}},
		out: []IntPoint{{s, 1.75, 1.5, 2}}},
	{in: []RawPoint{{s + -20, 1}, {s + 20, 2}},
		out: []IntPoint{{s, 1.75, 1.5, 2}, {s, 2, 2, 2}}},
	{in: []RawPoint{{s + -20, 1}, {s + 40, 4}},
		out: []IntPoint{{s, 2.5, 2, 3}, {s, 3.5, 3, 4}, {s, 4, 4, 4}}},
	{in: []RawPoint{{s + -20, 1}, {s + 100, 2}},
		out: []IntPoint{
			{s, math.NaN(), math.NaN(), math.NaN()},
			{s, math.NaN(), math.NaN(), math.NaN()},
			{s, math.NaN(), math.NaN(), math.NaN()},
			{s, math.NaN(), math.NaN(), math.NaN()},
			{s, math.NaN(), math.NaN(), math.NaN()}, {s, 2, 2, 2}},
	},
	{in: []RawPoint{{e + 0, 1}, {e + 10, 2}}, out: []IntPoint{}},
	{in: []RawPoint{{e + -10, 1}, {e + 10, 2}},
		out: []IntPoint{{s, 0, 0, 0}, {s, 1.25, 1, 1.5}}},
	{in: []RawPoint{{e + -20, 1}, {e + 10, 4}},
		out: []IntPoint{{s, 0, 0, 0}, {s, 2, 1, 3}}},
	{in: []RawPoint{}, out: []IntPoint{}},
}

func TestInterpolateAbs(t *testing.T) {
	for i := range absTests {
		// fill output with NaN and set times
		exp := resNaNFill(absTests[i].out)
		// run the interpolation
		res, err := Raw(absTests[i].in, Absolute, start, start+count*step, step, 4*step)
		// shouldn't have gotten any error
		if err != nil {
			t.Fatal("Test", i, "unexpected error:", err)
		}
		// test we got the right result
		if !resEqual(res, exp) {
			t.Error("Test", i, "failed!\nInput:\n", absTests[i].in,
				"\nOutput:\n", fmtInt(res), "\nExpected:\n", fmtInt(exp))
		}
	}
}
