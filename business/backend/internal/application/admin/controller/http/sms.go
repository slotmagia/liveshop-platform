package http

import (
	"context"

	apisms "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/sms"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type SMSReaderController struct{ service service.SMS }

func NewSMSReader(application service.SMS) *SMSReaderController {
	return &SMSReaderController{service: application}
}

func (c *SMSReaderController) Drivers(ctx context.Context, _ *apisms.DriversReq) (*apisms.DriversRes, error) {
	definitions := c.service.SMSDrivers(ctx)
	response := make(apisms.DriversRes, 0, len(definitions))
	for _, definition := range definitions {
		fields := make([]apisms.DriverField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			fields = append(fields, apisms.DriverField{Key: field.Key, Label: field.Label, Type: string(field.Type), Required: field.Required, Secret: field.Secret, Placeholder: field.Placeholder, Help: field.Help})
		}
		response = append(response, apisms.DriverDefinition{Code: string(definition.Code), Name: definition.Name, Description: definition.Description, Fields: fields})
	}
	return &response, nil
}

func (c *SMSReaderController) ListChannels(ctx context.Context, req *apisms.ListChannelsReq) (*apisms.ListChannelsRes, error) {
	items, err := c.service.ListSMSChannels(ctx, smsmodel.ChannelFilter{Keyword: req.Keyword, Driver: smsmodel.Driver(req.Driver), Lifecycle: smsmodel.Lifecycle(req.Lifecycle)})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apisms.ListChannelsRes, 0, len(items))
	for _, item := range items {
		response = append(response, channelView(item))
	}
	return &response, nil
}

func (c *SMSReaderController) ListRegions(ctx context.Context, req *apisms.ListRegionsReq) (*apisms.ListRegionsRes, error) {
	items, err := c.service.ListSMSRegions(ctx, smsmodel.RegionFilter{Keyword: req.Keyword, Lifecycle: smsmodel.Lifecycle(req.Lifecycle)})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apisms.ListRegionsRes, 0, len(items))
	for _, item := range items {
		response = append(response, regionView(item))
	}
	return &response, nil
}

func (c *SMSReaderController) GetMerchantGrant(ctx context.Context, req *apisms.GetMerchantGrantReq) (*apisms.GetMerchantGrantRes, error) {
	item, err := c.service.GetSMSMerchantGrant(ctx, req.MerchantID, req.ShopID)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.GetMerchantGrantRes(grantView(item))
	return &response, nil
}

type SMSWriterController struct{ service service.SMS }

func NewSMSWriter(application service.SMS) *SMSWriterController {
	return &SMSWriterController{service: application}
}

func (c *SMSWriterController) PutChannel(ctx context.Context, req *apisms.PutChannelReq) (*apisms.PutChannelRes, error) {
	secrets := make(map[string]smsmodel.CredentialChange, len(req.Secrets))
	for key, change := range req.Secrets {
		secrets[key] = smsmodel.CredentialChange{Mode: smsmodel.SecretMode(change.Mode), Value: change.Value}
	}
	item, err := c.service.PutSMSChannel(ctx, appmodel.PutSMSChannel{
		Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Name: req.Name,
		Driver: smsmodel.Driver(req.Driver), Region: req.Region, Priority: req.Priority, PublicConfig: req.PublicConfig, Secrets: secrets,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.PutChannelRes(channelView(item))
	return &response, nil
}

func (c *SMSWriterController) EnableChannel(ctx context.Context, req *apisms.EnableChannelReq) (*apisms.EnableChannelRes, error) {
	item, err := c.service.SetSMSChannelEnabled(ctx, appmodel.SetSMSEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: true})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.EnableChannelRes(channelView(item))
	return &response, nil
}

func (c *SMSWriterController) DisableChannel(ctx context.Context, req *apisms.DisableChannelReq) (*apisms.DisableChannelRes, error) {
	item, err := c.service.SetSMSChannelEnabled(ctx, appmodel.SetSMSEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: false})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.DisableChannelRes(channelView(item))
	return &response, nil
}

func (c *SMSWriterController) RetireChannel(ctx context.Context, req *apisms.RetireChannelReq) (*apisms.RetireChannelRes, error) {
	item, err := c.service.RetireSMSChannel(ctx, appmodel.RetireSMS{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.RetireChannelRes(channelView(item))
	return &response, nil
}

func (c *SMSWriterController) TestSend(ctx context.Context, req *apisms.TestSendReq) (*apisms.TestSendRes, error) {
	item, err := c.service.TestSMSSend(ctx, appmodel.TestSMSSend{ChannelCode: req.ChannelCode, Phone: req.Phone})
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apisms.TestSendRes{OK: item.OK, Detail: item.Detail, ChannelCode: item.ChannelCode, Driver: string(item.Driver), Mock: item.Mock, Code: item.Code}, nil
}

func (c *SMSWriterController) PutRegion(ctx context.Context, req *apisms.PutRegionReq) (*apisms.PutRegionRes, error) {
	item, err := c.service.PutSMSRegion(ctx, appmodel.PutSMSRegion{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, DialCode: req.DialCode, Name: req.Name, ISO2: req.ISO2, Emoji: req.Emoji, Sort: req.Sort})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.PutRegionRes(regionView(item))
	return &response, nil
}

func (c *SMSWriterController) EnableRegion(ctx context.Context, req *apisms.EnableRegionReq) (*apisms.EnableRegionRes, error) {
	item, err := c.service.SetSMSRegionEnabled(ctx, appmodel.SetSMSEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: true})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.EnableRegionRes(regionView(item))
	return &response, nil
}

func (c *SMSWriterController) DisableRegion(ctx context.Context, req *apisms.DisableRegionReq) (*apisms.DisableRegionRes, error) {
	item, err := c.service.SetSMSRegionEnabled(ctx, appmodel.SetSMSEnabled{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: false})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.DisableRegionRes(regionView(item))
	return &response, nil
}

func (c *SMSWriterController) RetireRegion(ctx context.Context, req *apisms.RetireRegionReq) (*apisms.RetireRegionRes, error) {
	item, err := c.service.RetireSMSRegion(ctx, appmodel.RetireSMS{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.RetireRegionRes(regionView(item))
	return &response, nil
}

func (c *SMSWriterController) PutMerchantGrant(ctx context.Context, req *apisms.PutMerchantGrantReq) (*apisms.PutMerchantGrantRes, error) {
	item, err := c.service.PutSMSMerchantGrant(ctx, appmodel.PutSMSMerchantGrant{CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, MerchantID: req.MerchantID, ShopID: req.ShopID, DialCodes: req.DialCodes})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apisms.PutMerchantGrantRes(grantView(item))
	return &response, nil
}

func channelView(item smsmodel.Channel) apisms.Channel {
	return apisms.Channel{
		ID: item.ID, Code: item.Code, Name: item.Name, Driver: string(item.Driver), Region: item.Region, Priority: item.Priority,
		Enabled: item.Enabled, Lifecycle: string(item.Lifecycle), PublicConfig: item.PublicConfig, SecretMasks: item.SecretMasks,
		CredentialKeyID: item.CredentialKeyID, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func regionView(item smsmodel.Region) apisms.Region {
	return apisms.Region{ID: item.ID, Code: item.Code, DialCode: item.DialCode, Name: item.Name, ISO2: item.ISO2, Emoji: item.Emoji, Sort: item.Sort, Enabled: item.Enabled, Lifecycle: string(item.Lifecycle), Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func grantView(item smsmodel.MerchantGrant) apisms.MerchantGrant {
	return apisms.MerchantGrant{ID: item.ID, MerchantID: item.MerchantID, ShopID: item.ShopID, DialCodes: item.DialCodes, Unrestricted: item.Unrestricted, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
