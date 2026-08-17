package model

type PushTransport string
type ConfigFieldType string

const (
	PushOBSRtmp    PushTransport = "OBS_RTMP"
	PushBrowserSDK PushTransport = "BROWSER_SDK"

	FieldText     ConfigFieldType = "TEXT"
	FieldPassword ConfigFieldType = "PASSWORD"
	FieldNumber   ConfigFieldType = "NUMBER"
	FieldSelect   ConfigFieldType = "SELECT"
)

const (
	CredentialSecret             = "SECRET"
	CredentialAppCertificate     = "APP_CERTIFICATE"
	CredentialCustomerCredential = "CUSTOMER_CREDENTIAL"
)

type ConfigOption struct {
	Value string
	Label string
}

// ConfigField is the driver-owned Admin form contract. Credential identifies
// fields that are write-only and therefore need KEEP/REPLACE/CLEAR semantics.
type ConfigField struct {
	Key         string
	Label       string
	Type        ConfigFieldType
	Group       string
	Required    bool
	Secret      bool
	Credential  string
	Default     string
	Placeholder string
	Help        string
	Options     []ConfigOption
	Min         int64
	Max         int64
	Advanced    bool
}

type DriverDefinition struct {
	Code          Driver
	Name          string
	Kind          Kind
	PushTransport PushTransport
	Description   string
	Fields        []ConfigField
}

var builtInDriverDefinitions = []DriverDefinition{
	{
		Code: DriverStatic, Name: "无鉴权直出", Kind: KindRTMP, PushTransport: PushOBSRtmp,
		Description: "开发态或内网 RTMP 直出；不签名，不能撤销已建立的推流连接。",
		Fields: []ConfigField{
			{Key: "app", Label: "RTMP App 名", Type: FieldText, Group: "RTMP 连接", Required: true, Default: "live", Placeholder: "live"},
			{Key: "pushDomain", Label: "推流域名", Type: FieldText, Group: "RTMP 连接", Placeholder: "push.example.com"},
			{Key: "pullDomain", Label: "拉流域名", Type: FieldText, Group: "RTMP 连接", Placeholder: "pull.example.com"},
		},
	},
	{
		Code: DriverSRS, Name: "自建 SRS", Kind: KindRTMP, PushTransport: PushOBSRtmp,
		Description: "OBS 推 RTMP，推拉流 URL 使用 HMAC-SHA256 Secret 与过期时间签名。",
		Fields:      append(rtmpSignedFields(), ConfigField{Key: "ttlSeconds", Label: "签名有效期（秒）", Type: FieldNumber, Group: "签名", Required: true, Default: "7200", Min: 60, Max: 2592000}),
	},
	{
		Code: DriverCloud, Name: "云直播", Kind: KindRTMP, PushTransport: PushOBSRtmp,
		Description: "OBS 推 RTMP，使用云直播风格的 txSecret/txTime 签名地址。",
		Fields:      append(rtmpSignedFields(), ConfigField{Key: "ttlSeconds", Label: "签名有效期（秒）", Type: FieldNumber, Group: "签名", Required: true, Default: "7200", Min: 60, Max: 2592000}),
	},
	{
		Code: DriverAgora, Name: "Agora RTC", Kind: KindRTC, PushTransport: PushBrowserSDK,
		Description: "主播和观众均使用浏览器 RTC SDK；服务端本地签发频道 Token。",
		Fields: []ConfigField{
			{Key: "agoraAppId", Label: "Agora App ID", Type: FieldText, Group: "RTC Token", Required: true, Placeholder: "Agora 项目 App ID", Help: "公开标识，来源于 Agora 项目管理页。"},
			{Key: "appCertificate", Label: "App Certificate", Type: FieldPassword, Group: "RTC Token", Secret: true, Credential: CredentialAppCertificate, Placeholder: "Agora 项目 App Certificate", Help: "只用于本地 Token 签名，与 REST Customer 凭据无关。"},
			{Key: "codec", Label: "RTC 视频编码", Type: FieldSelect, Group: "RTC Token", Required: true, Default: "vp8", Options: codecOptions("vp8", "h264", "vp9", "av1", "h265")},
			{Key: "customerKey", Label: "Customer Key", Type: FieldPassword, Group: "REST API（可选）", Secret: true, Credential: CredentialCustomerCredential, Placeholder: "RESTful API Customer Key"},
			{Key: "customerSecret", Label: "Customer Secret", Type: FieldPassword, Group: "REST API（可选）", Secret: true, Credential: CredentialCustomerCredential, Placeholder: "RESTful API Customer Secret", Help: "成对配置后可调用 Agora API 踢出残留会话。"},
			{Key: "ttlSeconds", Label: "Token 有效期（秒）", Type: FieldNumber, Group: "RTC Token", Required: true, Default: "7200", Min: 60, Max: 2592000},
		},
	},
	{
		Code: DriverAgoraMediaGateway, Name: "Agora Media Gateway", Kind: KindRTC, PushTransport: PushOBSRtmp,
		Description: "OBS 推 RTMP Ingest、观众 RTC 拉流；StreamKey 通过 Agora REST API 现签。",
		Fields: []ConfigField{
			{Key: "agoraAppId", Label: "Agora App ID", Type: FieldText, Group: "RTC Token", Required: true, Placeholder: "Agora 项目 App ID"},
			{Key: "appCertificate", Label: "App Certificate", Type: FieldPassword, Group: "RTC Token", Secret: true, Credential: CredentialAppCertificate, Placeholder: "Agora 项目 App Certificate"},
			{Key: "codec", Label: "Media Gateway 视频编码", Type: FieldSelect, Group: "Media Gateway", Required: true, Default: "h264", Options: codecOptions("h264", "h265"), Help: "Media Gateway 仅支持 H.264 / H.265 输入。"},
			{Key: "customerKey", Label: "Customer Key", Type: FieldPassword, Group: "REST API（必需）", Required: true, Secret: true, Credential: CredentialCustomerCredential, Placeholder: "RESTful API Customer Key"},
			{Key: "customerSecret", Label: "Customer Secret", Type: FieldPassword, Group: "REST API（必需）", Required: true, Secret: true, Credential: CredentialCustomerCredential, Placeholder: "RESTful API Customer Secret", Help: "用于现签和删除 Ingest StreamKey，必须成对配置。"},
			{Key: "region", Label: "Media Gateway 区域", Type: FieldSelect, Group: "Media Gateway", Required: true, Default: "na", Options: []ConfigOption{{Value: "na", Label: "北美 na"}, {Value: "eu", Label: "欧洲 eu"}, {Value: "ap", Label: "亚太 ap"}, {Value: "cn", Label: "中国 cn"}}},
			{Key: "ingestDomain", Label: "自定义 Ingest 域名", Type: FieldText, Group: "Media Gateway", Advanced: true, Placeholder: "留空则按 Region 自动推导", Help: "默认 rtls-ingress-prod-{region}.agoramdn.com:1935。"},
			{Key: "ttlSeconds", Label: "StreamKey/Token 有效期（秒）", Type: FieldNumber, Group: "Media Gateway", Required: true, Default: "7200", Min: 60, Max: 2592000},
		},
	},
}

func rtmpSignedFields() []ConfigField {
	return []ConfigField{
		{Key: "app", Label: "RTMP App 名", Type: FieldText, Group: "RTMP 连接", Required: true, Default: "live", Placeholder: "live"},
		{Key: "pushDomain", Label: "推流域名", Type: FieldText, Group: "RTMP 连接", Placeholder: "push.example.com"},
		{Key: "pullDomain", Label: "拉流域名", Type: FieldText, Group: "RTMP 连接", Placeholder: "pull.example.com"},
		{Key: "secret", Label: "签名 Secret", Type: FieldPassword, Group: "签名", Secret: true, Credential: CredentialSecret, Placeholder: "媒体服务器签名密钥"},
	}
}

func codecOptions(values ...string) []ConfigOption {
	options := make([]ConfigOption, 0, len(values))
	for _, value := range values {
		label := map[string]string{"vp8": "VP8", "h264": "H.264", "vp9": "VP9", "av1": "AV1", "h265": "H.265"}[value]
		options = append(options, ConfigOption{Value: value, Label: label})
	}
	return options
}

func DriverDefinitions() []DriverDefinition {
	definitions := make([]DriverDefinition, len(builtInDriverDefinitions))
	for index, definition := range builtInDriverDefinitions {
		definitions[index] = definition
		definitions[index].Fields = append([]ConfigField(nil), definition.Fields...)
		for fieldIndex := range definitions[index].Fields {
			definitions[index].Fields[fieldIndex].Options = append([]ConfigOption(nil), definition.Fields[fieldIndex].Options...)
		}
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
