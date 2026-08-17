package http

import (
	"context"

	apiemail "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/email"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type EmailReaderController struct{ service service.Email }

func NewEmailReader(application service.Email) *EmailReaderController {
	return &EmailReaderController{service: application}
}

func (c *EmailReaderController) Drivers(ctx context.Context, _ *apiemail.DriversReq) (*apiemail.DriversRes, error) {
	definitions := c.service.EmailDrivers(ctx)
	response := make(apiemail.DriversRes, 0, len(definitions))
	for _, definition := range definitions {
		fields := make([]apiemail.DriverField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			options := make([]apiemail.DriverFieldOption, 0, len(field.Options))
			for _, option := range field.Options {
				options = append(options, apiemail.DriverFieldOption{Value: option.Value, Label: option.Label})
			}
			fields = append(fields, apiemail.DriverField{Key: field.Key, Label: field.Label, Type: string(field.Type), Required: field.Required, Secret: field.Secret, Placeholder: field.Placeholder, Help: field.Help, Options: options})
		}
		response = append(response, apiemail.DriverDefinition{Code: string(definition.Code), Name: definition.Name, Description: definition.Description, Fields: fields})
	}
	return &response, nil
}

func (c *EmailReaderController) GetConfig(ctx context.Context, _ *apiemail.GetConfigReq) (*apiemail.GetConfigRes, error) {
	item, err := c.service.GetEmailConfig(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiemail.GetConfigRes(emailView(item))
	return &response, nil
}

type EmailWriterController struct{ service service.Email }

func NewEmailWriter(application service.Email) *EmailWriterController {
	return &EmailWriterController{service: application}
}

func (c *EmailWriterController) PutConfig(ctx context.Context, req *apiemail.PutConfigReq) (*apiemail.PutConfigRes, error) {
	secrets := map[string]emailmodel.CredentialChange{}
	for key, change := range req.Secrets {
		secrets[key] = emailmodel.CredentialChange{Mode: emailmodel.SecretMode(change.Mode), Value: change.Value}
	}
	item, err := c.service.PutEmailConfig(ctx, appmodel.PutEmailConfig{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Driver: emailmodel.Driver(req.Driver), PublicConfig: req.PublicConfig, Secrets: secrets,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiemail.PutConfigRes(emailView(item))
	return &response, nil
}

func (c *EmailWriterController) Enable(ctx context.Context, req *apiemail.EnableReq) (*apiemail.EnableRes, error) {
	item, err := c.service.SetEmailEnabled(ctx, appmodel.SetEmailEnabled{CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: true})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiemail.EnableRes(emailView(item))
	return &response, nil
}

func (c *EmailWriterController) Disable(ctx context.Context, req *apiemail.DisableReq) (*apiemail.DisableRes, error) {
	item, err := c.service.SetEmailEnabled(ctx, appmodel.SetEmailEnabled{CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Enabled: false})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiemail.DisableRes(emailView(item))
	return &response, nil
}

func (c *EmailWriterController) TestSend(ctx context.Context, req *apiemail.TestSendReq) (*apiemail.TestSendRes, error) {
	item, err := c.service.TestEmailSend(ctx, appmodel.TestEmailSend{To: req.To, Subject: req.Subject})
	if err != nil {
		return nil, web.Failure(err)
	}
	return &apiemail.TestSendRes{OK: item.OK, Detail: item.Detail, Driver: string(item.Driver), Mock: item.Mock}, nil
}

func emailView(item emailmodel.Config) apiemail.Config {
	return apiemail.Config{
		ID: item.ID, Driver: string(item.Driver), Enabled: item.Enabled, PublicConfig: item.PublicConfig, SecretMasks: item.SecretMasks,
		CredentialKeyID: item.CredentialKeyID, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
