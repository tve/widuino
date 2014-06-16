// Interpolate time series data
// Copyright (c) 2014 by Thorsten von Eicken

package interpol8

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"math"
)

// Raw data point, serves as the input to the interpolation. The value can either be
// the absolute value at the point in time, or it can be the delta count (derivative)
// from the previous point to the current one. The unit of the timestamp does not affect
// the interpolation, so it's OK to use seconds since 1970, milliseconds since 1970, or
// just about anything else that fits.
type RawPoint struct {
	Asof  uint64  // typically seconds since 1970 or millisecs, doesn't matter
	Value float64 // data value at that point in time
}

// The kind of data points indicated whether we're dealing with absolute (gauge)
// values or whether we're dealing with counter/rate (derivative) values
type Kind int

const (
	Absolute = iota
	Rate
)

// Interpolated data point, output from the interpolation. Each point represents an interval
// *starting* at the timestamp.
type IntPoint struct {
	Asof          uint64
	Avg, Min, Max float64
}

// Interpolate an array of raw datapoints to produce evenly spaced data points from start to
// end in step sized intervals. If step doesn't divide evenly into (end-start) the end is
// pushed out.
// Assumes that the incoming raw array is sorted by time and that it includes one data point
// before the start of the interval to produce and one data point past it. More precisely,
// raw[0].Asof <= start and raw[last].Asof >= end+step. If no data point pre- or
// post-interval is provided then it is assumed to be NaN.
// The maxFill value determines how large an interval the interpolator is allowed to
// interpolate over before it has to produce NaN values. Specifically, if raw values
// are <= maxFill time apart then interpolation will happen but if they're further apart
// then NaNs will be produced. maxFill must be at least equal to step.
func Raw(raw []RawPoint, kind Kind, start, end, step uint64, maxFill uint64) ([]IntPoint, error) {
	//fmt.Printf("interpol8.Raw: requested %d..%d by %d\n", start, end, step)
	// Input validation
	if end <= start {
		return nil, errors.New("interpol8.Raw: end <= start")
	}
	if step < 2 { // need step>=2 so step/2 is meaningful
		return nil, errors.New("interpol8.Raw: step < 2")
	}
	if maxFill < step {
		return nil, errors.New("interpol8.Raw: maxFill < step")
	}
	// verify that raw is sorted; TODO: verify that duplicates are OK
	for i := 0; i < len(raw)-1; i++ {
		if raw[i].Asof > raw[i+1].Asof {
			return nil, errors.New("interpol8.Raw: raw input is not sorted")
		}
	}

	// count datapoints to produce and adjust ending
	count, end, err := adjustEnd(start, end, step)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("interpol8.Raw: producing %d..%d by %d, returning %d points\n",
	//	start, end, step, count)
	glog.V(2).Infof("interpol8.Raw: producing %d..%d by %d, returning %d points",
		start, end, step, count)

	// trim data points we don't need
	raw = lTrim(raw, start, step)
	raw = rTrim(raw, end, step)

	// Handle no-data case
	if len(raw) == 0 {
		return makeNaNs(start, step, count), nil
	}

	// Iterate through all the points we need to produce and calculate the value of each one
	// based on the raw points that fall within (and around) the result point interval.
	res := make([]IntPoint, count, count)
	for i, r, ts := -1, 0, start; ts < end; r, ts = r+1, ts+step {
		//fmt.Printf("i=%d ts=%d r=%d\n", i, ts, r)
		// r is index into result slice
		// i is index into raw
		// ts is start of res[r] interval
		te := ts + step // te is end of res[r] interval

		// make i the last point before ts, that's the first point we need
		// to interpolate from
		for i+1 < len(raw) && raw[i+1].Asof < ts {
			i++
		}
		// now either raw[i+1] >= ts or i+1==len(raw)
		// and either i<0 or raw[i] < ts

		// find the last point before or equal to ts+step,
		// that's the last point we need to interpolate to
		j := i
		for j+1 < len(raw) && raw[j+1].Asof <= te {
			j++
		}

		// interpolate between raw[i]&raw[i+1] up to raw[j]&raw[j+1]
		num := 0.0   // integral under the piece-wise linear "curve"
		denom := 0.0 // timestamp range covered, in the end the step's value is num/denom
		min := math.NaN()
		max := math.NaN()
		for x := i; x <= j; x++ {
			if x < 0 {
				continue // no current point, nothing to do
			}
			// raw[x] & raw[x+1] exist, interpolate area between them
			tx := raw[x].Asof
			vx := raw[x].Value
			// handle min/max for raw[x]
			if kind == Absolute && tx >= ts && tx <= te {
				if math.IsNaN(min) || vx < min {
					min = vx
				}
				if math.IsNaN(max) || vx > max {
					max = vx
				}
			}
			// check whether there is a next point
			if x+1 >= len(raw) {
				continue
			}
			tx1 := raw[x+1].Asof
			// calculate the rate from raw[x] to raw[x+1], we may need it
			var raw_rate float64
			if tx1 > tx {
				if kind == Rate {
					raw_rate = (raw[x+1].Value - raw[x].Value) / float64(tx1-tx)

					if raw_rate < 0 {
						// we don't support negative rates, we take negative rates
						// to indicate a counter reset or roll-over, in which case
						// we don't really know what the rate has been
						raw_rate = math.NaN()
					}
				}
			}

			// handle avg and interpolation
			switch {

			// the first point in the interval is at the boundary
			case tx1 == ts:
				// nothing to interpolate...

			// the last point in the interval is at the boundary
			case tx == te:
				// nothing to interpolate...

				// zero distance
			case tx == tx1:
				// can't interpolate (not entirely satisfactory)

			// both points inside interval
			case tx >= ts && tx1 <= te:
				dx := float64(tx1 - tx)
				denom += dx
				switch kind {
				case Absolute:
					num += dx * (raw[x+1].Value + raw[x].Value) / 2
				case Rate:
					if raw[x+1].Value >= raw[x].Value {
						// negative rate => rate unknown
						num += raw[x+1].Value - raw[x].Value
						if math.IsNaN(min) || raw_rate < min {
							min = raw_rate
						}
						if math.IsNaN(max) || raw_rate > max {
							max = raw_rate
						}
					}
				}

			// points too far apart to interpolate across
			case tx1-tx > maxFill:
				// nothing to interpolate...

			// raw[x] is before and raw[x+1] is inside the interval
			case tx < ts && tx1 <= te:
				dx := float64(tx1 - ts)
				denom += dx
				ratio := float64(ts-tx) / float64(tx1-tx)
				iy := raw[x].Value + ratio*(raw[x+1].Value-raw[x].Value)
				switch kind {
				case Absolute:
					num += dx * (raw[x+1].Value + iy) / 2
					if math.IsNaN(min) || iy < min {
						min = iy
					}
					if math.IsNaN(max) || iy > max {
						max = iy
					}
				case Rate:
					if raw[x+1].Value >= iy {
						// negative rate => rate unknown
						num += raw[x+1].Value - iy
						if math.IsNaN(min) || raw_rate < min {
							min = raw_rate
						}
						if math.IsNaN(max) || raw_rate > max {
							max = raw_rate
						}
					}
				}

			// raw[x] is inside and raw[x+1] is beyond the interval
			case tx >= ts && tx1 > te:
				dx := float64(te - tx)
				denom += dx
				ratio := dx / float64(tx1-tx)
				iy := raw[x].Value + ratio*(raw[x+1].Value-raw[x].Value)
				switch kind {
				case Absolute:
					num += dx * (raw[x].Value + iy) / 2
					if math.IsNaN(min) || iy < min {
						min = iy
					}
					if math.IsNaN(max) || iy > max {
						max = iy
					}
				case Rate:
					if iy >= raw[x].Value {
						// negative rate => rate unknown
						num += iy - raw[x].Value
						if math.IsNaN(min) || raw_rate < min {
							min = raw_rate
						}
						if math.IsNaN(max) || raw_rate > max {
							max = raw_rate
						}
					}
				}

			// raw[x] is before and raw[x+1] is beyond the interval
			case tx < ts && tx1 > te:
				switch kind {
				case Absolute:
					dx := float64(te - ts)
					denom += dx
					ratio1 := float64(ts-tx) / float64(tx1-tx)
					ratio2 := float64(te-tx) / float64(tx1-tx)
					iy1 := raw[x].Value + ratio1*(raw[x+1].Value-raw[x].Value)
					iy2 := raw[x].Value + ratio2*(raw[x+1].Value-raw[x].Value)
					num += dx * (iy1 + iy2) / 2
					if math.IsNaN(min) || iy1 < min {
						min = iy1
					}
					if math.IsNaN(max) || iy1 > max {
						max = iy1
					}
					if math.IsNaN(min) || iy2 < min {
						min = iy2
					}
					if math.IsNaN(max) || iy2 > max {
						max = iy2
					}
				case Rate:
					if raw[x+1].Value >= raw[x].Value {
						// negative rate => rate unknown
						denom = float64(tx1 - tx)
						num = raw[x+1].Value - raw[x].Value

						if math.IsNaN(min) || raw_rate < min {
							min = raw_rate
						}
						if math.IsNaN(max) || raw_rate > max {
							max = raw_rate
						}
					}
				}
			default:
				return nil, errors.New(fmt.Sprint(
					"interpol8.Raw: Internal error! x=", x,
					" tx=", tx, " tx1=", tx1, " ts=", ts))
			}
		}

		// save what we've computed
		if denom > 0.0 {
			res[r] = IntPoint{Asof: ts, Avg: num / denom, Min: min, Max: max}
		} else {
			if kind == Absolute && i+1 < len(raw) &&
				raw[i+1].Asof >= ts && raw[i+1].Asof < te {
				// special-case: we have a single data point in the interval,
				// no neighbor to interpolate with -- use that data point as value
				v := raw[i+1].Value
				res[r] = IntPoint{Asof: ts, Avg: v, Min: v, Max: v}
			} else {
				nan := math.NaN()
				res[r] = IntPoint{Asof: ts, Avg: nan, Min: nan, Max: nan}
			}
		}
	} // end of aggregation/interpolation loop

	return res, nil
}

//===== Helper functions =====

// create result array filled with NaNs
func makeNaNs(start, step uint64, count int) []IntPoint {
	res := make([]IntPoint, count)
	nan := math.NaN()
	for i := range res {
		res[i].Asof = start + uint64(i)*step
		res[i].Min = nan
		res[i].Max = nan
		res[i].Avg = nan
	}
	return res
}

// adjust the end and calculate the number of result steps
func adjustEnd(start, end, step uint64) (int, uint64, error) {
	if end <= start {
		return 0, 0, errors.New("interpol8.Raw error: end <= start")
	}
	count := (end-1-start)/step + 1
	if count > 1000000 {
		return 0, 0, errors.New(
			"interpol8.Raw: will not produce more than one million datapoints")
	}
	end = start + count*step
	return int(count), end, nil
}

// eliminate raw datapoints before the start that we don't need
func lTrim(raw []RawPoint, start, step uint64) []RawPoint {
	for i := 1; i < len(raw); i++ {
		if raw[i].Asof > start {
			// raw[i] is inside the result range, thus we want to keep the previous
			// point outside the interval
			return raw[i-1:]
		}
	}
	return raw
}

// eliminate raw datapoints after the end that we don't need
func rTrim(raw []RawPoint, end, step uint64) []RawPoint {
	for i := len(raw) - 2; i > 0; i-- {
		if raw[i].Asof < end+step {
			// raw[i] is inside the result range, thus we want to keep the following
			// point outside the interval
			return raw[:i+2] // we want 0..(i+1) inclusive
		}
	}
	return raw
}
