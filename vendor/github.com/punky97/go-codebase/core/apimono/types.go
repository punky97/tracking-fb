package apimono

// DMSCodeName -- dms code name
type DMSCodeName string

func (x DMSCodeName) String() string {
	return string(x)
}

// GatewayCodeName -- gateway code name
type GatewayCodeName string

func (x GatewayCodeName) String() string {
	return string(x)
}

// APICodeName -- api code name
type APICodeName struct {
	CodeName     string
	HTTPBasePath string
}

func NewApiCodeName(codeName string, basePath string) APICodeName {
	return APICodeName{
		CodeName:     codeName,
		HTTPBasePath: basePath,
	}
}
