package domain

import "time"

// Withdrawal представляет одно списание баллов с накопительного счета.
type Withdrawal struct {
	// ID содержит внутренний идентификатор списания.
	ID int64

	// UserID содержит идентификатор пользователя, который выполнил списание.
	UserID int64

	// OrderNumber содержит номер заказа, в счет которого списаны баллы.
	OrderNumber string

	// Sum содержит сумму списания в минимальных целых единицах.
	Sum Points

	// ProcessedAt содержит время регистрации списания.
	ProcessedAt time.Time
}
