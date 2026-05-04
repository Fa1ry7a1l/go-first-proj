// Package luhn проверяет числовые строки по контрольной сумме алгоритма Луна.
package luhn

// Valid сообщает, состоит ли номер только из цифр и проходит ли он проверку
// контрольной суммы по алгоритму Луна.
func Valid(number string) bool {
	if number == "" {
		return false
	}

	sum := 0
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		if number[i] < '0' || number[i] > '9' {
			return false
		}

		digit := int(number[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	return sum%10 == 0
}
