package model

type ConfigFieldType string

const (
	FieldText     ConfigFieldType = "TEXT"
	FieldPassword ConfigFieldType = "PASSWORD"
	FieldNumber   ConfigFieldType = "NUMBER"
	FieldSelect   ConfigFieldType = "SELECT"
)

type Driver string

const (
	DriverMock Driver = "mock"
	DriverSMTP Driver = "smtp"
)

type FieldOption struct {
	Value string
	Label string
}

type ConfigField struct {
	Key         string
	Label       string
	Type        ConfigFieldType
	Required    bool
	Secret      bool
	Placeholder string
	Help        string
	Options     []FieldOption
}

type DriverDefinition struct {
	Code        Driver
	Name        string
	Description string
	Fields      []ConfigField
}

var builtInDriverDefinitions = []DriverDefinition{
	{
		Code: DriverMock, Name: "Mock（开发/测试，不真发）",
		Description: "只记录测试发送，不连接 SMTP。本地和联调用此驱动验收发信配置。",
	},
	{
		Code: DriverSMTP, Name: "SMTP",
		Description: "按 host/port/encryption 连接 SMTP；密码只写，留空保留原值。",
		Fields: []ConfigField{
			{Key: "host", Label: "SMTP 服务器", Type: FieldText, Required: true, Placeholder: "smtp.exmail.qq.com"},
			{Key: "port", Label: "端口", Type: FieldNumber, Required: true, Placeholder: "465"},
			{Key: "encryption", Label: "加密方式", Type: FieldSelect, Required: true, Options: []FieldOption{
				{Value: "ssl", Label: "SSL（隐式 TLS，通常对应 465）"},
				{Value: "starttls", Label: "STARTTLS（通常对应 587/25）"},
				{Value: "none", Label: "不加密"},
			}},
			{Key: "username", Label: "账号", Type: FieldText, Required: true, Placeholder: "notify@example.com"},
			{Key: "password", Label: "密码/授权码", Type: FieldPassword, Required: true, Secret: true},
			{Key: "from_address", Label: "发件人地址", Type: FieldText, Placeholder: "notify@example.com", Help: "留空则默认与账号相同。"},
			{Key: "from_name", Label: "发件人名称", Type: FieldText, Help: "收件方看到的发件人显示名，留空则只显示地址。"},
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
