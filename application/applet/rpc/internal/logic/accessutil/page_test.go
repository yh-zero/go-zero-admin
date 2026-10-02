package accessutil

import (
	"math"
	"testing"
)

func TestPageRejectsOffsetOverflow(t *testing.T) {
	for _, input := range [][2]int64{{0, 10}, {-1, 10}, {1, 0}, {1, 501}, {math.MaxInt64, 500}, {math.MinInt64, 1}} {
		if offset, limit, err := Page(input[0], input[1]); err == nil || offset != 0 || limit != 0 {
			t.Fatalf("invalid page %v accepted: offset=%d limit=%d err=%v", input, offset, limit, err)
		}
	}
	maxOffset := int64(^uint(0) >> 1)
	lastPage := maxOffset/500 + 1
	offset, limit, err := Page(lastPage, 500)
	if err != nil || int64(offset) != (lastPage-1)*500 || limit != 500 {
		t.Fatalf("last representable offset failed: %d %d %v", offset, limit, err)
	}
	if _, _, err := Page(lastPage+1, 500); err == nil {
		t.Fatal("offset beyond the platform int range accepted")
	}
	if offset, limit, err := Page(2, 10); err != nil || offset != 10 || limit != 10 {
		t.Fatal("ordinary pagination changed")
	}
}
