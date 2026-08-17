package model

type ConfigFieldType string

const (
	FieldText     ConfigFieldType = "TEXT"
	FieldPassword ConfigFieldType = "PASSWORD"
)

type Driver string

const (
	DriverLocal        Driver = "local"
	DriverAliyunOSS    Driver = "aliyun_oss"
	DriverCloudflareR2 Driver = "cloudflare_r2"
)

type ConfigField struct {
	Key         string
	Label       string
	Type        ConfigFieldType
	Required    bool
	Secret      bool
	Placeholder string
	Help        string
}

type DriverDefinition struct {
	Code        Driver
	Name        string
	Description string
	Fields      []ConfigField
}

var builtInDriverDefinitions = []DriverDefinition{
	{
		Code: DriverLocal, Name: "本地磁盘",
		Description: "写入 Platform 进程本地目录。路径是服务器文件系统属性，不经表单改写；适合单机和联调。",
	},
	{
		Code: DriverAliyunOSS, Name: "阿里云 OSS",
		Description: "PUT Object + OSS V1 签名；对象按 public-read 写入。",
		Fields: []ConfigField{
			{Key: "endpoint", Label: "Endpoint", Type: FieldText, Required: true, Placeholder: "oss-cn-hangzhou.aliyuncs.com"},
			{Key: "bucket", Label: "Bucket", Type: FieldText, Required: true},
			{Key: "access_key_id", Label: "AccessKey ID", Type: FieldText, Required: true},
			{Key: "access_key_secret", Label: "AccessKey Secret", Type: FieldPassword, Required: true, Secret: true},
			{Key: "public_base_url", Label: "自定义访问域名（选填）", Type: FieldText, Help: "留空则用 https://{bucket}.{endpoint}"},
		},
	},
	{
		Code: DriverCloudflareR2, Name: "Cloudflare R2",
		Description: "S3 兼容 PUT + AWS SigV4。桶默认不公开，必须填写已开通的公开访问域名。",
		Fields: []ConfigField{
			{Key: "account_id", Label: "Account ID", Type: FieldText, Required: true},
			{Key: "bucket", Label: "Bucket", Type: FieldText, Required: true},
			{Key: "access_key_id", Label: "Access Key ID", Type: FieldText, Required: true},
			{Key: "secret_access_key", Label: "Secret Access Key", Type: FieldPassword, Required: true, Secret: true},
			{Key: "public_base_url", Label: "公开访问域名", Type: FieldText, Required: true, Placeholder: "https://pub-xxxx.r2.dev 或自定义域名", Help: "需在 Cloudflare 控制台为该桶开启 Public access 或绑定自定义域名。"},
		},
	},
}

func DriverDefinitions() []DriverDefinition {
	definitions := make([]DriverDefinition, len(builtInDriverDefinitions))
	for index, definition := range builtInDriverDefinitions {
		definitions[index] = definition
		definitions[index].Fields = append([]ConfigField(nil), definition.Fields...)
	}
	return definitions
}

func DefinitionFor(driver Driver) (DriverDefinition, bool) {
	for _, definition := range builtInDriverDefinitions {
		if definition.Code == driver {
			return definition, true
		}
	}
	return DriverDefinition{}, false
}

func SecretFieldKeys(driver Driver) []string {
	definition, ok := DefinitionFor(driver)
	if !ok {
		return nil
	}
	keys := make([]string, 0)
	for _, field := range definition.Fields {
		if field.Secret {
			keys = append(keys, field.Key)
		}
	}
	return keys
}
