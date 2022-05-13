package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type Service struct {
	url    url.URL
	client *resty.Client
}

type OrderStatus struct {
	Number  string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual"`
}

func New(address string) (Service, error) {
	if address == "" {
		return Service{}, ErrConfigInvalidAddress
	}
	u, err := url.Parse(address)
	if err != nil {
		return Service{}, err
	}
	return Service{
		url:    *u,
		client: resty.New(),
	}, nil
}

func (s Service) CheckOrder(number string) (OrderStatus, error) {
	req, endpoint := s.prepareRequest("/api/orders/%s", number)
	resp, err := req.Get(endpoint)
	if err != nil {
		return OrderStatus{}, err
	}
	switch resp.StatusCode() {
	case http.StatusNoContent:
		return OrderStatus{}, ErrOrderNotFound
	case http.StatusTooManyRequests:
		retryAfterVal := resp.Header().Get("Retry-After")
		retryAfter, convErr := strconv.Atoi(retryAfterVal)
		if convErr != nil || retryAfter < 0 {
			return OrderStatus{}, ErrRespInvalidWaitTime
		}
		return OrderStatus{}, NewErrTooManyRequests(uint(retryAfter))
	case http.StatusOK:
		var os OrderStatus
		if jsonErr := json.Unmarshal(resp.Body(), &os); jsonErr != nil {
			log.Warn().Err(jsonErr).Str("order", number).Msg("Unable to parse json response for 200 OK")
			return OrderStatus{}, ErrRespInvalidData
		}
		return os, nil
	default:
		return OrderStatus{}, ErrRespInvalidStatus
	}
}

func (s Service) prepareRequest(uri string, args ...interface{}) (*resty.Request, string) {
	endpoint := s.url
	endpoint.Path = fmt.Sprintf(uri, args...)
	req := s.client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json")
	return req, endpoint.String()
}
