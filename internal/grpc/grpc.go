// Package grpc описывает методы работы через grpc
package grpc

import (
	"context"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	pb "github.com/colzphml/yandex_project/internal/metrics/proto"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/rs/zerolog/log"
)

func ConvertGRPCtoMetric(in *pb.Metric) (metrics.Metrics, error) {
	metric := metrics.Metrics{
		ID:    in.Id,
		MType: in.Mtype,
		Hash:  in.Hash,
	}
	switch in.Mtype {
	case "gauge":
		value := in.Value
		metric.Value = &value
	case "counter":
		value := in.Delta
		metric.Delta = &value
	default:
		return metrics.Metrics{}, metrics.ErrWrongType
	}
	return metric, nil
}

func ConvertMetrictoGRPC(in metrics.Metrics) *pb.Metric {
	var value float64
	if in.Value != nil {
		value = *in.Value
	}
	var delta int64
	if in.Delta != nil {
		delta = *in.Delta
	}
	result := pb.Metric{
		Id:    in.ID,
		Mtype: in.MType,
		Value: value,
		Delta: delta,
		Hash:  in.Hash,
	}
	return &result
}

type MetricsServer struct {
	pb.UnimplementedMetricsServer
	Repo storage.Repositorier
	Cfg  *serverutils.ServerConfig
}

func (s *MetricsServer) Save(ctx context.Context, in *pb.SaveMetricRequest) (*pb.SaveMetricResponse, error) {
	var resp pb.SaveMetricResponse
	metric, err := ConvertGRPCtoMetric(in.Metric)
	if err != nil {
		resp.Error = err.Error()
		return &resp, nil
	}
	compareHash, err := metric.CompareHash(s.Cfg.Key)
	if err != nil {
		resp.Error = err.Error()
		return &resp, nil
	}
	if !compareHash {
		resp.Error = "signature is wrong"
		return &resp, nil
	}
	err = s.Repo.SaveMetric(ctx, metric)
	if err != nil {
		log.Error().Err(err).Str("can't save metric", metric.ID)
		resp.Error = err.Error()
		return &resp, nil
	}
	if s.Cfg.StoreInterval.Nanoseconds() == 0 {
		err = s.Repo.DumpMetrics(ctx, s.Cfg)
		if err != nil {
			log.Error().Err(err).Str("can't store metric", metric.ID)
			resp.Error = err.Error()
			return &resp, nil
		}
	}
	return &resp, nil

}
func (s *MetricsServer) SaveList(ctx context.Context, in *pb.SaveListMetricsRequest) (*pb.SaveListMetricsResponse, error) {
	var ms []metrics.Metrics
	var resp pb.SaveListMetricsResponse
	for _, v := range in.Metric {
		m, err := ConvertGRPCtoMetric(v)
		if err != nil {
			resp.Error = err.Error()
			return &resp, nil
		}
		ms = append(ms, m)
	}
	for _, v := range ms {
		compareHash, err := v.CompareHash(s.Cfg.Key)
		if err != nil {
			resp.Error = err.Error()
			return &resp, nil
		}
		if !compareHash {
			resp.Error = "signature is wrong"
			return &resp, nil
		}
	}
	_, err := s.Repo.SaveListMetric(ctx, ms)
	if err != nil {
		log.Error().Err(err).Msg("can't save metrics")
		resp.Error = err.Error()
		return &resp, nil
	}
	if s.Cfg.StoreInterval.Nanoseconds() == 0 {
		err = s.Repo.DumpMetrics(ctx, s.Cfg)
		if err != nil {
			log.Error().Err(err).Msg("can't store metric")
			resp.Error = err.Error()
			return &resp, nil
		}
	}
	return &resp, nil
}

func (s *MetricsServer) Get(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	var resp pb.GetMetricResponse
	metricValue, err := s.Repo.GetValue(ctx, in.MetricName)
	if err != nil {
		resp.Error = err.Error()
		return &resp, nil
	}
	resp.Metric = &pb.Metric{
		Id:    metricValue.ID,
		Mtype: metricValue.MType,
		Value: *metricValue.Value,
		Delta: *metricValue.Delta,
	}
	return &resp, nil
}

func (s *MetricsServer) GetList(ctx context.Context, in *pb.GetListMetricRequest) (*pb.GetListMetricResponse, error) {
	var resp pb.GetListMetricResponse
	var result []*pb.Metric
	metricList := s.Repo.ListMetrics(ctx)
	for _, v := range metricList {
		result = append(result, ConvertMetrictoGRPC(v))
	}
	resp.Metric = result
	return &resp, nil
}

func (s *MetricsServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	var resp pb.PingResponse
	err := s.Repo.Ping(ctx)
	if err != nil {
		resp.Ping = false
		return &resp, nil
	}
	resp.Ping = true
	return &resp, nil
}
