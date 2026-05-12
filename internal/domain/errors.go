// Package domain содержит основные бизнес-типы и доменные ошибки приложения.
package domain

import "errors"

// Доменные ошибки возвращаются сервисами и реализациями хранилища, чтобы HTTP-слой
// мог преобразовать их в корректные коды ответа.
var (
	// ErrInvalidCredentials означает, что логин или пароль пользователя неверны.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserAlreadyExists означает, что пользователь с таким логином уже зарегистрирован.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrUserInvalidCredentialsFormat означает, что логин или пароль отсутствуют в запросе.
	ErrUserInvalidCredentialsFormat = errors.New("invalid credentials format")

	// ErrUserNotFound означает, что пользователь с указанным логином отсутствует в хранилище.
	ErrUserNotFound = errors.New("user not found")

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

	// ErrInsufficientFunds означает, что у пользователя недостаточно баллов для списания.
	ErrInsufficientFunds = errors.New("insufficient funds")

	// ErrWithdrawalInvalidSum означает, что сумма списания не является положительным числом.
	ErrWithdrawalInvalidSum = errors.New("withdrawal sum is invalid")
)
