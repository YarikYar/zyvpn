package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ExchangeRates struct {
	TONUSD float64 `json:"ton_usd"`
	USDRUB float64 `json:"usd_rub"`
	TONRUB float64 `json:"ton_rub"`
}

type RatesService struct {
	cache     *ExchangeRates
	cacheMu   sync.RWMutex
	cacheTime time.Time
	cacheTTL  time.Duration
}

func NewRatesService() *RatesService {
	return &RatesService{
		cacheTTL: 5 * time.Minute,
	}
}

func (s *RatesService) GetRates() (*ExchangeRates, error) {
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		rates := *s.cache
		s.cacheMu.RUnlock()
		return &rates, nil
	}
	s.cacheMu.RUnlock()

	// Fetch fresh rates
	rates, err := s.fetchRates()
	if err != nil {
		// Return cached if available
		s.cacheMu.RLock()
		if s.cache != nil {
			cached := *s.cache
			s.cacheMu.RUnlock()
			return &cached, nil
		}
		s.cacheMu.RUnlock()
		return nil, err
	}

	s.cacheMu.Lock()
	s.cache = rates
	s.cacheTime = time.Now()
	s.cacheMu.Unlock()

	return rates, nil
}

func (s *RatesService) fetchRates() (*ExchangeRates, error) {
	rates := &ExchangeRates{}

	// Fetch TON/USD from STON.fi API
	tonUSD, err := s.fetchTONRate()
	if err != nil {
		// Fallback
		tonUSD = 5.0
	}
	rates.TONUSD = tonUSD

	// Fetch USD/RUB from exchangerate.host (free, no API key)
	usdRUB, err := s.fetchRUBRate()
	if err != nil {
		// Fallback
		usdRUB = 95.0
	}
	rates.USDRUB = usdRUB

	rates.TONRUB = rates.TONUSD * rates.USDRUB

	return rates, nil
}

func (s *RatesService) fetchTONRate() (float64, error) {
	// Use CoinGecko API (free, no key required)
	resp, err := http.Get("https://api.coingecko.com/api/v3/simple/price?ids=the-open-network&vs_currencies=usd")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		TON struct {
			USD float64 `json:"usd"`
		} `json:"the-open-network"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.TON.USD == 0 {
		return 0, fmt.Errorf("invalid TON rate")
	}

	return result.TON.USD, nil
}

func (s *RatesService) fetchRUBRate() (float64, error) {
	// Use exchangerate.host API (free)
	resp, err := http.Get("https://api.exchangerate.host/latest?base=USD&symbols=RUB")
	if err != nil {
		// Try alternative API
		return s.fetchRUBRateFallback()
	}
	defer resp.Body.Close()

	var result struct {
		Rates struct {
			RUB float64 `json:"RUB"`
		} `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.fetchRUBRateFallback()
	}

	if result.Rates.RUB == 0 {
		return s.fetchRUBRateFallback()
	}

	return result.Rates.RUB, nil
}

func (s *RatesService) fetchRUBRateFallback() (float64, error) {
	// Use CBR (Central Bank of Russia) API
	resp, err := http.Get("https://www.cbr-xml-daily.ru/daily_json.js")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Valute struct {
			USD struct {
				Value float64 `json:"Value"`
			} `json:"USD"`
		} `json:"Valute"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Valute.USD.Value == 0 {
		return 0, fmt.Errorf("invalid RUB rate")
	}

	return result.Valute.USD.Value, nil
}
