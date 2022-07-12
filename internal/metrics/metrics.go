package metrics

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrUndefinedType = errors.New("type of metric undefined")
	ErrParseMetric   = errors.New("can't parse metric")
	ErrWrongType     = errors.New("metric have another type")
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func (m *Metrics) ValueString() string {
	switch m.MType {
	case "gauge":
		return strconv.FormatFloat(float64(*m.Value), 'g', -1, 64)
	case "counter":
		return strconv.FormatInt(int64(*m.Delta), 10)
	default:
		return ""
	}
}

func (m *Metrics) CalculateHash(key string) ([]byte, error) {
	var src string
	switch m.MType {
	case "gauge":
		src = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case "counter":
		src = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	default:
		return nil, ErrUndefinedType
	}
	hash, err := signData(src, key)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (m *Metrics) FillHash(key string) error {
	if key != "" {
		hash, err := m.CalculateHash(key)
		if err != nil {
			return err
		}
		m.Hash = hex.EncodeToString(hash)
	}
	return nil
}

func (m *Metrics) CompareHash(key string) (bool, error) {
	if key != "" {
		hash, err := m.CalculateHash(key)
		if err != nil {
			return false, err
		}
		data, err := hex.DecodeString(m.Hash)
		if err != nil {
			return false, err
		}
		return bytes.Equal(hash, data), nil
	}
	return true, nil
}

func signData(src, key string) ([]byte, error) {
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(src))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
