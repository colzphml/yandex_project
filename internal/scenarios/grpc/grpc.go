// Package grpc описывает методы работы через grpc
package grpc

import (
	"context"
	"errors"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	pb "github.com/colzphml/yandex_project/internal/metrics/proto"
	"github.com/colzphml/yandex_project/internal/scenarios"
	"github.com/colzphml/yandex_project/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errMapping(err error) codes.Code {
	switch {
	case errors.Is(err, scenarios.ErrStatusBadRequest):
		return codes.InvalidArgument
	case errors.Is(err, scenarios.ErrStatusNotFound):
		return codes.NotFound
	case errors.Is(err, scenarios.ErrStatusNotImplemented):
		return codes.Unimplemented
	default:
		return codes.Internal
	}
}

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
		return nil, status.Error(codes.Internal, err.Error())
	}
	err = scenarios.SaveMetric(ctx, s.Repo, s.Cfg, metric, true)
	if err != nil {
		return nil, status.Error(errMapping(err), err.Error())
	}
	return &resp, nil

}
func (s *MetricsServer) SaveList(ctx context.Context, in *pb.SaveListMetricsRequest) (*pb.SaveListMetricsResponse, error) {
	var ms []metrics.Metrics
	var resp pb.SaveListMetricsResponse
	for _, v := range in.Metric {
		m, err := ConvertGRPCtoMetric(v)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		ms = append(ms, m)
	}
	_, err := scenarios.SaveArrayMetric(ctx, s.Repo, s.Cfg, ms)
	if err != nil {
		return nil, status.Error(errMapping(err), err.Error())
	}
	return &resp, nil
}

func (s *MetricsServer) Get(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	var resp pb.GetMetricResponse
	metricValue, err := s.Repo.GetValue(ctx, in.MetricName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
