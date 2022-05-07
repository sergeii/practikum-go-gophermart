package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
)

type Service struct {
	url url.URL
}

type OrderStatus struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
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
		url: *u,
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
		if jsonErr := json.Unmarshal(resp.Body(), &os); err != nil {
			return OrderStatus{}, jsonErr
		}
		return os, nil
	default:
		return OrderStatus{}, ErrRespInvalidStatus
	}
}

func (s Service) prepareRequest(uri string, args ...interface{}) (*resty.Request, string) {
	endpoint := s.url
	endpoint.Path = fmt.Sprintf(uri, args...)
	client := resty.New()
	req := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json")
	return req, endpoint.String()
}
