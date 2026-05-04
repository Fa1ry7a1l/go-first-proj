package domain

// PointsScale задает количество минимальных единиц в одном балле.
const PointsScale int64 = 100

// Points хранит баллы в минимальных целых единицах, чтобы избежать ошибок
// округления при начислениях и списаниях.
type Points int64

// Float64 возвращает представление баллов для JSON-ответов на границе HTTP API.
func (p Points) Float64() float64 {
	return float64(p) / float64(PointsScale)
}
