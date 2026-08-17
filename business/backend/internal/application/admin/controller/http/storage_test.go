package http

import (
	"context"
	"testing"

	apistorage "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/storage"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

type storageServiceStub struct {
	put appmodel.PutStorageChannel
}

func (s *storageServiceStub) StorageDrivers(context.Context) []storagemodel.DriverDefinition {
	return nil
}
func (s *storageServiceStub) ListStorageChannels(context.Context, storagemodel.ChannelFilter) ([]storagemodel.Channel, error) {
	return nil, nil
}
func (s *storageServiceStub) PutStorageChannel(_ context.Context, input appmodel.PutStorageChannel) (storagemodel.Channel, error) {
	s.put = input
	return storagemodel.Channel{Code: input.Code, Enabled: true, PublicConfig: input.PublicConfig}, nil
}
func (s *storageServiceStub) SetStorageChannelEnabled(context.Context, appmodel.SetStorageEnabled) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *storageServiceStub) SetStorageDefault(context.Context, appmodel.SetStorageDefault) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *storageServiceStub) RetireStorageChannel(context.Context, appmodel.RetireStorage) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *storageServiceStub) TestStorageChannel(context.Context, appmodel.TestStorageChannel) (storagemodel.TestResult, error) {
	return storagemodel.TestResult{}, nil
}
func (s *storageServiceStub) GetStorageObject(context.Context, string) (storagemodel.Object, error) {
	return storagemodel.Object{}, nil
}

func TestStoragePutChannelKeepsSecretsWriteOnly(t *testing.T) {
	service := &storageServiceStub{}
	controller := NewStorageWriter(service)
	response, err := controller.PutChannel(context.Background(), &apistorage.PutChannelReq{
		Code: "oss-cn", Name: "杭州 OSS", Driver: "aliyun_oss",
		PublicConfig: map[string]string{"endpoint": "oss-cn-hangzhou.aliyuncs.com", "bucket": "demo", "access_key_id": "id"},
		Secrets:      map[string]apistorage.CredentialChange{"access_key_secret": {Mode: "REPLACE", Value: "plain-secret"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.put.Code != "oss-cn" || service.put.Secrets["access_key_secret"].Value != "plain-secret" {
		t.Fatalf("put=%#v", service.put)
	}
	if response.SecretMasks != nil && response.SecretMasks["access_key_secret"] == "plain-secret" {
		t.Fatalf("response leaked secret: %#v", response)
	}
}
