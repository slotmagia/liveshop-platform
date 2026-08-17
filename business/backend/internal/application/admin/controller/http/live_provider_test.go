package http

import (
	"context"
	"testing"

	apiprovider "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
)

type liveProviderServiceStub struct {
	putCalls int
	putInput appmodel.PutLiveProvider
}

func (s *liveProviderServiceStub) LiveProviderDrivers(context.Context) []providermodel.DriverDefinition {
	return nil
}

func (s *liveProviderServiceStub) ListLiveProviders(context.Context, providermodel.Filter) ([]providermodel.Provider, error) {
	return nil, nil
}

func (s *liveProviderServiceStub) PutLiveProvider(_ context.Context, input appmodel.PutLiveProvider) (providermodel.Provider, error) {
	s.putCalls++
	s.putInput = input
	return providermodel.Provider{Code: input.Code, Enabled: true}, nil
}

func (s *liveProviderServiceStub) RetireLiveProvider(context.Context, appmodel.RetireLiveProvider) (providermodel.Provider, error) {
	return providermodel.Provider{}, nil
}

func (s *liveProviderServiceStub) GetLiveProviderAssignments(context.Context, int64) (providermodel.AssignmentSet, error) {
	return providermodel.AssignmentSet{}, nil
}

func (s *liveProviderServiceStub) PutLiveProviderAssignments(context.Context, appmodel.PutLiveProviderAssignments) (providermodel.AssignmentSet, error) {
	return providermodel.AssignmentSet{}, nil
}

func TestLiveProviderPutDoesNotRequireEnabledParameter(t *testing.T) {
	service := &liveProviderServiceStub{}
	controller := NewLiveProviderWriter(service)

	response, err := controller.Put(context.Background(), &apiprovider.PutReq{Code: "srs-main"})
	if err != nil {
		t.Fatalf("request without enabled parameter was rejected: %v", err)
	}
	if service.putCalls != 1 || !response.Enabled {
		t.Fatalf("request was not forwarded: calls=%d response=%#v", service.putCalls, response)
	}
}
