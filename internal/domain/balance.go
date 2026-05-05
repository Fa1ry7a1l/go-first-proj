package domain

// Balance описывает текущее состояние накопительного счета пользователя.
type Balance struct {
	// Current содержит доступный остаток баллов в минимальных целых единицах.
	Current Points

	// Withdrawn содержит сумму всех списаний в минимальных целых единицах.
	Withdrawn Points
}
