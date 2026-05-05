package domain

import "testing"

func TestPointsFloat64(t *testing.T) {
	points := Points(12345)

	if got := points.Float64(); got != 123.45 {
		t.Fatalf("Float64 = %v, want 123.45", got)
	}
}

func TestPointsFromFloat64(t *testing.T) {
	points := PointsFromFloat64(123.456)

	if points != 12346 {
		t.Fatalf("PointsFromFloat64 = %d, want 12346", points)
	}
}
