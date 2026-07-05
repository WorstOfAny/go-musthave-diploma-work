package service

import(
	"strconv"
	"unicode/utf8"
	"github.com/rs/zerolog/log"
)

type serviceError string

func (se serviceError) Error() string {
	return string(se)
}

const ErrFormatOrderNum = serviceError("bad number format")
const ErrParseOrderNum = serviceError("failed to parse num from string")


func CheckOrderNum(num string) error {
	parsedNum, err := strconv.Atoi(num)
	ndig := utf8.RuneCountInString(num)

	if err != nil {
		log.Error().Err(err).Msgf("failed to parse order num: %v", err)
		return ErrParseOrderNum
	}

	var sum int
	tmpNum := parsedNum

	for i := 1; i <= ndig; i++ {
		val := tmpNum % 10
		tmpNum /= 10

		if i % 2 == 0 {
			val *= 2
			if val > 9 { val = val - 9 }
		}

		sum += val
	}

	if sum % 10 != 0 { return ErrFormatOrderNum }

	return nil
}
