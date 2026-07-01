package models

import(
	"time"
	"encoding/json"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/service"
)

type WithdrawRequest struct {
	UID *int `json:"-" db:"user_id"`
	Order string `json:"order" db:"order_number"`
	Sum *float64 `json:"sum" db:"sum_value"`
	ProcessedAt time.Time `json:"processed_at"`
}

func (wr *WithdrawRequest) UnmarshalJSON(data []byte) error {
	
	type Alias WithdrawRequest

	al := &struct{
		*Alias
	}{
		Alias: (*Alias)(wr),
	}

	if err := json.Unmarshal(data, al); err != nil { return err }
	err := service.CheckOrderNum(wr.Order)
	if err != nil { return err }

	return nil
}
