package http

import (
	"context"
	"testing"

	apiemail "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/email"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
)

type emailServiceStub struct {
	put appmodel.PutEmailConfig
}

func (s *emailServiceStub) EmailDrivers(context.Context) []emailmodel.DriverDefinition { return nil }
func (s *emailServiceStub) GetEmailConfig(context.Context) (emailmodel.Config, error) {
	return emailmodel.Config{}, nil
}
func (s *emailServiceStub) PutEmailConfig(_ context.Context, input appmodel.PutEmailConfig) (emailmodel.Config, error) {
	s.put = input
	return emailmodel.Config{Driver: input.Driver, Enabled: true, PublicConfig: input.PublicConfig}, nil
}
func (s *emailServiceStub) SetEmailEnabled(context.Context, appmodel.SetEmailEnabled) (emailmodel.Config, error) {
	return emailmodel.Config{}, nil
}
func (s *emailServiceStub) TestEmailSend(context.Context, appmodel.TestEmailSend) (emailmodel.TestSendResult, error) {
	return emailmodel.TestSendResult{}, nil
}

func TestEmailPutKeepsSecretsWriteOnly(t *testing.T) {
	service := &emailServiceStub{}
	controller := NewEmailWriter(service)
	response, err := controller.PutConfig(context.Background(), &apiemail.PutConfigReq{
		Driver:  "mock",
		Secrets: map[string]apiemail.CredentialChange{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.put.Driver != "mock" || !response.Enabled || len(response.SecretMasks) != 0 {
		t.Fatalf("put=%#v response=%#v", service.put, response)
	}
}
