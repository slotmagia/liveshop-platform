package http

import (
	"context"

	apiprovider "github.com/liveshop-platform/module-platform/internal/application/admin/api/http/v1/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type LiveProviderReaderController struct{ service service.LiveProvider }

func NewLiveProviderReader(application service.LiveProvider) *LiveProviderReaderController {
	return &LiveProviderReaderController{service: application}
}

func (c *LiveProviderReaderController) Drivers(ctx context.Context, _ *apiprovider.DriversReq) (*apiprovider.DriversRes, error) {
	definitions := c.service.LiveProviderDrivers(ctx)
	response := make(apiprovider.DriversRes, 0, len(definitions))
	for _, definition := range definitions {
		fields := make([]apiprovider.DriverField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			options := make([]apiprovider.DriverOption, 0, len(field.Options))
			for _, option := range field.Options {
				options = append(options, apiprovider.DriverOption{Value: option.Value, Label: option.Label})
			}
			fields = append(fields, apiprovider.DriverField{
				Key: field.Key, Label: field.Label, Type: string(field.Type), Group: field.Group,
				Required: field.Required, Secret: field.Secret, Credential: field.Credential,
				Default: field.Default, Placeholder: field.Placeholder, Help: field.Help,
				Options: options, Min: field.Min, Max: field.Max, Advanced: field.Advanced,
			})
		}
		response = append(response, apiprovider.DriverDefinition{
			Code: string(definition.Code), Name: definition.Name, Kind: string(definition.Kind),
			PushTransport: string(definition.PushTransport), Description: definition.Description, Fields: fields,
		})
	}
	return &response, nil
}

func (c *LiveProviderReaderController) List(ctx context.Context, req *apiprovider.ListReq) (*apiprovider.ListRes, error) {
	items, err := c.service.ListLiveProviders(ctx, providermodel.Filter{Keyword: req.Keyword, Kind: providermodel.Kind(req.Kind), Driver: providermodel.Driver(req.Driver), Lifecycle: providermodel.Lifecycle(req.Lifecycle)})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := make(apiprovider.ListRes, 0, len(items))
	for _, item := range items {
		response = append(response, providerView(item))
	}
	return &response, nil
}

type LiveProviderWriterController struct{ service service.LiveProvider }

func NewLiveProviderWriter(application service.LiveProvider) *LiveProviderWriterController {
	return &LiveProviderWriterController{service: application}
}

func (c *LiveProviderWriterController) Put(ctx context.Context, req *apiprovider.PutReq) (*apiprovider.PutRes, error) {
	item, err := c.service.PutLiveProvider(ctx, appmodel.PutLiveProvider{
		Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, Name: req.Name,
		Driver: providermodel.Driver(req.Driver), App: req.App, PushDomain: req.PushDomain, PullDomain: req.PullDomain,
		AgoraAppID: req.AgoraAppID, Codec: req.Codec, IngestDomain: req.IngestDomain, Region: req.Region,
		TTLSeconds: req.TTLSeconds, IsDefault: req.IsDefault,
		Secret: credentialChange(req.Secret), AppCertificate: credentialChange(req.AppCertificate), CustomerCredential: credentialChange(req.CustomerCredential),
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiprovider.PutRes(providerView(item))
	return &response, nil
}

func (c *LiveProviderReaderController) Assignments(ctx context.Context, req *apiprovider.GetAssignmentsReq) (*apiprovider.GetAssignmentsRes, error) {
	item, err := c.service.GetLiveProviderAssignments(ctx, req.MerchantID)
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiprovider.GetAssignmentsRes(assignmentView(item))
	return &response, nil
}

func (c *LiveProviderWriterController) PutAssignments(ctx context.Context, req *apiprovider.PutAssignmentsReq) (*apiprovider.PutAssignmentsRes, error) {
	providers := make([]appmodel.LiveProviderAssignment, 0, len(req.Providers))
	for _, item := range req.Providers {
		providers = append(providers, appmodel.LiveProviderAssignment{ProviderCode: item.ProviderCode, Name: item.Name, Enabled: item.Enabled, Default: item.Default})
	}
	item, err := c.service.PutLiveProviderAssignments(ctx, appmodel.PutLiveProviderAssignments{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, MerchantID: req.MerchantID, Providers: providers,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiprovider.PutAssignmentsRes(assignmentView(item))
	return &response, nil
}

func assignmentView(item providermodel.AssignmentSet) apiprovider.AssignmentSet {
	out := apiprovider.AssignmentSet{MerchantID: item.MerchantID, Providers: []apiprovider.Assignment{}, Version: item.Version}
	for _, grant := range item.Providers {
		out.Providers = append(out.Providers, apiprovider.Assignment{ProviderCode: grant.ProviderCode, Name: grant.Name, Enabled: grant.Enabled, Default: grant.Default})
	}
	return out
}

func (c *LiveProviderWriterController) Retire(ctx context.Context, req *apiprovider.RetireReq) (*apiprovider.RetireRes, error) {
	item, err := c.service.RetireLiveProvider(ctx, appmodel.RetireLiveProvider{Code: req.Code, CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		return nil, web.Failure(err)
	}
	response := apiprovider.RetireRes(providerView(item))
	return &response, nil
}

func credentialChange(input apiprovider.CredentialChange) providermodel.CredentialChange {
	return providermodel.CredentialChange{Mode: providermodel.SecretMode(input.Mode), Value: input.Value, SecondaryValue: input.SecondaryValue}
}

func providerView(item providermodel.Provider) apiprovider.Provider {
	return apiprovider.Provider{
		ID: item.ID, Code: item.Code, Name: item.Name, Kind: string(item.Kind), Driver: string(item.Driver), App: item.App,
		PushDomain: item.PushDomain, PullDomain: item.PullDomain, AgoraAppID: item.AgoraAppID, Codec: item.Codec,
		IngestDomain: item.IngestDomain, Region: item.Region, TTLSeconds: item.TTLSeconds,
		Enabled: item.Enabled, IsDefault: item.IsDefault, Lifecycle: string(item.Lifecycle), HealthStatus: string(item.Health),
		HealthMessage: item.HealthMessage, HealthCheckedAt: item.HealthCheckedAt,
		SecretSet: item.Credentials.SecretSet, SecretMask: item.Credentials.SecretMask,
		AppCertificateSet: item.Credentials.AppCertificateSet, AppCertificateMask: item.Credentials.AppCertificateMask,
		CustomerCredentialSet: item.Credentials.CustomerCredentialSet, CustomerKeyMask: item.Credentials.CustomerKeyMask,
		CustomerSecretMask: item.Credentials.CustomerSecretMask, CredentialKeyID: item.Credentials.KeyID,
		Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
