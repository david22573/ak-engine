package data

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ObjectFilter struct{}

func NewObjectFilter() *ObjectFilter {
	return &ObjectFilter{}
}

func (f *ObjectFilter) Filter(objects []ObjectStats, req CandleRequest) ([]ObjectStats, error) {
	var matched []ObjectStats
	for _, obj := range objects {
		var start, end time.Time
		if obj.MinOpenTimeMS != 0 && obj.MaxOpenTimeMS != 0 {
			start = time.UnixMilli(obj.MinOpenTimeMS)
			end = time.UnixMilli(obj.MaxOpenTimeMS)
		} else {
			var err error
			start, end, err = ParseDateRangeFromFilename(obj.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to determine date range for object %s: %w", obj.Key, err)
			}
		}

		if !req.From.IsZero() && end.Before(req.From) {
			continue
		}
		if !req.To.IsZero() && start.After(req.To) {
			continue
		}
		matched = append(matched, obj)
	}
	return matched, nil
}

var dateRegex = regexp.MustCompile(`\d{4}-\d{2}(?:-\d{2})?`)

func ParseDateRangeFromFilename(path string) (time.Time, time.Time, error) {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, ".parquet")

	matches := dateRegex.FindAllString(filename, -1)
	if len(matches) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no date match found in filename %s", filename)
	}

	dateStr := matches[len(matches)-1]
	parts := strings.Split(dateStr, "-")

	if len(parts) == 3 {
		// Daily: YYYY-MM-DD
		start, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end := start.AddDate(0, 0, 1).Add(-time.Millisecond)
		return start, end, nil
	} else if len(parts) == 2 {
		// Monthly: YYYY-MM
		start, err := time.Parse("2006-01", dateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end := start.AddDate(0, 1, 0).Add(-time.Millisecond)
		return start, end, nil
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unsupported date format in filename %s", filename)
}
