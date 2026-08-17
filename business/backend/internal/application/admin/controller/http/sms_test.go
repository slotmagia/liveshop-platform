package http

import (
	"context"
	"testing"

	apisms "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/sms"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
)

type smsServiceStub struct {
	put appmodel.PutSMSChannel
}

func (s *smsServiceStub) SMSDrivers(context.Context) []smsmodel.DriverDefinition { return nil }
func (s *smsServiceStub) ListSMSChannels(context.Context, smsmodel.ChannelFilter) ([]smsmodel.Channel, error) {
	return nil, nil
}
func (s *smsServiceStub) PutSMSChannel(_ context.Context, input appmodel.PutSMSChannel) (smsmodel.Channel, error) {
	s.put = input
	return smsmodel.Channel{Code: input.Code, Enabled: true, PublicConfig: input.PublicConfig}, nil
}
func (s *smsServiceStub) SetSMSChannelEnabled(context.Context, appmodel.SetSMSEnabled) (smsmodel.Channel, error) {
	return smsmodel.Channel{}, nil
}
func (s *smsServiceStub) RetireSMSChannel(context.Context, appmodel.RetireSMS) (smsmodel.Channel, error) {
	return smsmodel.Channel{}, nil
}
func (s *smsServiceStub) ListSMSRegions(context.Context, smsmodel.RegionFilter) ([]smsmodel.Region, error) {
	return nil, nil
}
func (s *smsServiceStub) PutSMSRegion(context.Context, appmodel.PutSMSRegion) (smsmodel.Region, error) {
	return smsmodel.Region{}, nil
}
func (s *smsServiceStub) SetSMSRegionEnabled(context.Context, appmodel.SetSMSEnabled) (smsmodel.Region, error) {
	return smsmodel.Region{}, nil
}
func (s *smsServiceStub) RetireSMSRegion(context.Context, appmodel.RetireSMS) (smsmodel.Region, error) {
	return smsmodel.Region{}, nil
}
func (s *smsServiceStub) GetSMSMerchantGrant(context.Context, int64, int64) (smsmodel.MerchantGrant, error) {
	return smsmodel.MerchantGrant{}, nil
}
func (s *smsServiceStub) PutSMSMerchantGrant(context.Context, appmodel.PutSMSMerchantGrant) (smsmodel.MerchantGrant, error) {
	return smsmodel.MerchantGrant{}, nil
}
func (s *smsServiceStub) TestSMSSend(context.Context, appmodel.TestSMSSend) (smsmodel.TestSendResult, error) {
	return smsmodel.TestSendResult{}, nil
}

func TestSMSPutChannelKeepsSecretsWriteOnly(t *testing.T) {
	service := &smsServiceStub{}
	controller := NewSMSWriter(service)
	response, err := controller.PutChannel(context.Background(), &apisms.PutChannelReq{
		Code: "mock-dev", Name: "开发通道", Driver: "mock", Region: "*",
		Secrets: map[string]apisms.CredentialChange{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.put.Code != "mock-dev" || !response.Enabled {
		t.Fatalf("put=%#v response=%#v", service.put, response)
	}
}
