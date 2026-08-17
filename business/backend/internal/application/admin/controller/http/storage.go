package http

import (
	"context"

	apistorage "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/storage"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type StorageReaderController struct{ service service.Storage }

func NewStorageReader(application service.Storage) *StorageReaderController {
	return &StorageReaderController{service: application}
}

func (c *StorageReaderController) Drivers(ctx context.Context, _ *apistorage.DriversReq) (*apistorage.DriversRes, error) {
	definitions := c.service.StorageDrivers(ctx)
	response := make(apistorage.DriversRes, 0, len(definitions))
	for _, definition := range definitions {
		fields := make([]apistorage.DriverField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			fields = append(fields, apistorage.DriverField{Key: field.Key, Label: field.Label, Type: string(field.Type), Required: field.Required, Secret: field.Secret, Placeholder: field.Placeholder, Help: field.Help})
		}
		response = append(response, apistorage.DriverDefinition{Code: string(definition.Code), Name: definition.Name, Description: definition.Description, Fields: fields})
	}
	return &response, nil
}

func (c *StorageReaderController) ListChannels(ctx context.Context, req *apistorage.ListChannelsReq) (*apistorage.ListChannelsRes, error) {
	items, err := c.service.ListStorageChannels(ctx, storagemodel.ChannelFilter{Keyword: req.Keyword, Driver: storagemodel.Driver(req.Driver), Lifecycle: storagemodel.Lifecycle(req.Lifecycle)})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apistorage.ListChannelsRes, 0, len(items))
	for _, item := range items {
		response = append(response, storageChannelView(item))
	}
	return &response, nil
}

type StorageWriterController struct{ service service.Storage }

func NewStorageWriter(application service.Storage) *StorageWriterController {
	return &StorageWriterController{service: application}
}

func (c *StorageWriterController) PutChannel(ctx context.Context, req *apistorage.PutChannelReq) (*apistorage.PutChannelRes, error) {
	secrets := make(map[string]storagemodel.CredentialChange, len(req.Secrets))
	for key, change := range req.Secrets {
		secrets[key] = storagemodel.CredentialChange{Mode: storagemodel.SecretMode(change.Mode), Value: change.Value}
	}
	item, err := c.service.PutStorageChannel(ctx, appmodel.PutStorageChannel{
		Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Name: req.Name,
		Driver: storagemodel.Driver(req.Driver), PublicConfig: req.PublicConfig, Secrets: secrets,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apistorage.PutChannelRes(storageChannelView(item))
	return &response, nil
}

func (c *StorageWriterController) EnableChannel(ctx context.Context, req *apistorage.EnableChannelReq) (*apistorage.EnableChannelRes, error) {
	item, err := c.service.SetStorageChannelEnabled(ctx, appmodel.SetStorageEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: true})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apistorage.EnableChannelRes(storageChannelView(item))
	return &response, nil
}

func (c *StorageWriterController) DisableChannel(ctx context.Context, req *apistorage.DisableChannelReq) (*apistorage.DisableChannelRes, error) {
	item, err := c.service.SetStorageChannelEnabled(ctx, appmodel.SetStorageEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: false})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apistorage.DisableChannelRes(storageChannelView(item))
	return &response, nil
}

func (c *StorageWriterController) SetDefault(ctx context.Context, req *apistorage.SetDefaultReq) (*apistorage.SetDefaultRes, error) {
	item, err := c.service.SetStorageDefault(ctx, appmodel.SetStorageDefault{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apistorage.SetDefaultRes(storageChannelView(item))
	return &response, nil
}

func (c *StorageWriterController) RetireChannel(ctx context.Context, req *apistorage.RetireChannelReq) (*apistorage.RetireChannelRes, error) {
	item, err := c.service.RetireStorageChannel(ctx, appmodel.RetireStorage{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apistorage.RetireChannelRes(storageChannelView(item))
	return &response, nil
}

func (c *StorageWriterController) TestChannel(ctx context.Context, req *apistorage.TestChannelReq) (*apistorage.TestChannelRes, error) {
	item, err := c.service.TestStorageChannel(ctx, appmodel.TestStorageChannel{Code: req.Code})
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apistorage.TestChannelRes{OK: item.OK, Detail: item.Detail, URL: item.URL, Driver: string(item.Driver)}, nil
}

func storageChannelView(item storagemodel.Channel) apistorage.Channel {
	return apistorage.Channel{
		ID: item.ID, Code: item.Code, Name: item.Name, Driver: string(item.Driver), Enabled: item.Enabled, IsDefault: item.IsDefault,
		Lifecycle: string(item.Lifecycle), PublicConfig: item.PublicConfig, SecretMasks: item.SecretMasks,
		CredentialKeyID: item.CredentialKeyID, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
