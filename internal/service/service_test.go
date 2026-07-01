package service

import(
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCheckOrderNum(t *testing.T) {
	type want struct {
		returnError bool
		err serviceError
	}

	testCases := []struct{
		name string
		num string
		want want
	}{
		{
			name: "correct number",
			num: "12345674",
			want: want{},
		},
		{
			name: "correct number",
			num: "7992738",
			want: want{},
		},
		{
			name: "correct number",
			num: "49927398716",
			want: want{},
		},
		{
			name: "bad format number",
			num: "12345675",
			want: want{ returnError: true, err: ErrFormatOrderNum },
		},
		{
			name: "bad number",
			num: "123s5675",
			want: want{ returnError: true, err: ErrParseOrderNum },
		},
		{
			name: "empty number",
			num: "",
			want: want{ returnError: true, err: ErrParseOrderNum },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckOrderNum(tc.num)

			if tc.want.returnError {
				assert.Error(t, err)
				assert.ErrorIs(t, tc.want.err, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
