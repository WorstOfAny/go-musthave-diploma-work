package client

import(
	"errors"
	"strconv"
	"time"
	"io"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"syscall"
	"net"
	"net/http"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	"encoding/json"
	"fmt"
)

// Клиент для получения информации от системы расчётов
type Client struct {
	client *resty.Client
}

// Создание клиента
func NewClient(baseURL string) *Client {
	restyC := resty.New()
	restyC.
		SetBaseURL(baseURL).
		SetTimeout(time.Second * 15).
		SetRetryCount(3).
		SetRetryMaxWaitTime(15 * time.Second).
		SetRetryAfter(func(c *resty.Client, r *resty.Response) (time.Duration, error) {
			delay := time.Duration(2 * (r.Request.Attempt) - 1) * time.Second
			if r.StatusCode() == http.StatusTooManyRequests {
				ra, err := strconv.Atoi(r.Header().Get("Retry-After"))
				if err != nil {
					return delay, err
				}
				delay = time.Duration(ra) * time.Second
			}
			return delay, nil
		}).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() { return true }
				return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF)
			}
			return r.StatusCode() == http.StatusTooManyRequests || r.StatusCode() > 499
		}).
		AddRetryHook(func(r *resty.Response, err error) {
			if err != nil { log.Info().Str("attempt", strconv.Itoa(r.Request.Attempt)).Err(err).Msg("Request attempt") }
		})
	return &Client{ client: restyC }
}

// Получение информации от системы расчётов
func (c *Client) Fetch(o *models.Order) error {
	resp, err := c.client.R().
		SetDoNotParseResponse(true).
		Get(o.Number)
	if err != nil {
		log.Error().Err(err).Msg("request err")
		return err
	}

	defer resp.RawResponse.Body.Close()

	if resp.StatusCode() > 299 {
		return fmt.Errorf("response failed with code: %d", resp.StatusCode())
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil
	}

		dec := json.NewDecoder(resp.RawResponse.Body)
		err = dec.Decode(o)
		if err != nil {
			return fmt.Errorf("failed parse json: %w", err)
		}

	return nil
}
