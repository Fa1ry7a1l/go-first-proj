// Package domain содержит основные бизнес-типы и доменные ошибки приложения.
package domain

import "errors"

// Доменные ошибки возвращаются сервисами и реализациями хранилища, чтобы HTTP-слой
// мог преобразовать их в корректные коды ответа.
var (
	// ErrOrderInvalidFormat означает, что номер заказа пустой или содержит нецифровые символы.
	ErrOrderInvalidFormat = errors.New("order number has invalid format")

	// ErrOrderInvalidNumber означает, что номер заказа не прошел проверку алгоритмом Луна.
	ErrOrderInvalidNumber = errors.New("order number is invalid")

	// ErrOrderAlreadyUploadedByUser означает, что этот пользователь уже загружал такой заказ.
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by this user")

	// ErrOrderUploadedByAnotherUser означает, что заказ уже привязан к другому пользователю.
	ErrOrderUploadedByAnotherUser = errors.New("order already uploaded by another user")

	// ErrOrderNotFound означает, что заказ с указанным номером отсутствует в хранилище.
	ErrOrderNotFound = errors.New("order not found")
)
