package model

type ConfigFieldType string

const (
	FieldText     ConfigFieldType = "TEXT"
	FieldPassword ConfigFieldType = "PASSWORD"
)

type Driver string

const (
	DriverMock    Driver = "mock"
	DriverAliyun  Driver = "aliyun"
	DriverYunpian Driver = "yunpian"
	DriverTwilio  Driver = "twilio"
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
		Code: DriverMock, Name: "Mock（开发/测试，不真发）",
		Description: "只记录测试发送，不调用外部网关。本地和联调用此驱动验收通道、区域和路由。",
	},
	{
		Code: DriverAliyun, Name: "阿里云短信 Aliyun（Dysmsapi）",
		Description: "调用阿里云 Dysmsapi SendSms；模板变量固定为 code。",
		Fields: []ConfigField{
			{Key: "access_key_id", Label: "AccessKey ID", Type: FieldText, Required: true},
			{Key: "access_key_secret", Label: "AccessKey Secret", Type: FieldPassword, Required: true, Secret: true},
			{Key: "sign_name", Label: "短信签名", Type: FieldText, Required: true, Placeholder: "店铺签名"},
			{Key: "template_code", Label: "模板 CODE", Type: FieldText, Required: true, Placeholder: "SMS_251035488", Help: "模板内验证码占位变量需命名为 code。"},
			{Key: "template_code@en-US", Label: "模板 CODE（en-US，可选）", Type: FieldText, Help: "英文收件人使用的阿里云模板；留空回退默认模板。"},
			{Key: "region_id", Label: "RegionId（可选）", Type: FieldText, Placeholder: "cn-hangzhou"},
		},
	},
	{
		Code: DriverYunpian, Name: "云片 Yunpian（国际）",
		Description: "POST 云片 single_send；文案用 {code} 表示验证码。",
		Fields: []ConfigField{
			{Key: "api_key", Label: "API Key", Type: FieldPassword, Required: true, Secret: true},
			{Key: "text_template", Label: "短信文案模板", Type: FieldText, Required: true, Placeholder: "Your verification code is {code}", Help: "用 {code} 表示验证码占位。"},
			{Key: "text_template@en-US", Label: "短信文案模板（en-US，可选）", Type: FieldText, Help: "英文收件人使用；留空回退默认模板。"},
			{Key: "endpoint", Label: "接口地址（可选）", Type: FieldText, Placeholder: "https://sms.yunpian.com/v2/sms/single_send.json"},
		},
	},
	{
		Code: DriverTwilio, Name: "Twilio（国际）",
		Description: "POST Twilio Messages.json；文案用 {code} 表示验证码。",
		Fields: []ConfigField{
			{Key: "account_sid", Label: "Account SID", Type: FieldText, Required: true},
			{Key: "auth_token", Label: "Auth Token", Type: FieldPassword, Required: true, Secret: true},
			{Key: "from", Label: "发送号码 From", Type: FieldText, Required: true, Placeholder: "+1xxxxxxxxxx"},
			{Key: "text_template", Label: "短信文案模板", Type: FieldText, Required: true, Placeholder: "Your verify code: {code}", Help: "用 {code} 表示验证码占位。"},
			{Key: "text_template@en-US", Label: "短信文案模板（en-US，可选）", Type: FieldText, Placeholder: "Your verify code: {code}"},
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
