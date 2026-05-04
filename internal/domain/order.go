package domain

import "time"

// OrderStatus описывает текущее состояние обработки начислений по заказу.
type OrderStatus string

const (
	// OrderStatusNew означает, что заказ сохранен, но еще не отправлен в обработку начислений.
	OrderStatusNew OrderStatus = "NEW"

	// OrderStatusProcessing означает, что внешняя система еще рассчитывает начисление.
	OrderStatusProcessing OrderStatus = "PROCESSING"

	// OrderStatusInvalid означает, что внешняя система отказала в начислении по заказу.
	OrderStatusInvalid OrderStatus = "INVALID"

	// OrderStatusProcessed означает, что расчет завершен и начисление, если оно есть, сохранено.
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// Order представляет заказ, который пользователь загрузил в систему лояльности.
type Order struct {
	// ID содержит внутренний идентификатор заказа в хранилище.
	ID int64

	// UserID содержит идентификатор пользователя, которому принадлежит заказ.
	UserID int64

	// Number содержит исходный номер заказа.
	Number string

	// Status содержит текущий статус обработки заказа.
	Status OrderStatus

	// Accrual содержит начисленные баллы в минимальных целых единицах, если расчет уже дал результат.
	Accrual *Points

	// UploadedAt содержит время загрузки заказа пользователем.
	UploadedAt time.Time

	// UpdatedAt содержит время последнего обновления заказа.
	UpdatedAt time.Time
}
