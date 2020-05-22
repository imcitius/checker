package config


//type ChatAlert interface {
//	SendAlert(alerttype string, e error) error
//	GetAlertName() string
//	GetAlertType() string
//	GetAlertCreds() string
//}

type IncomingChatMessage interface {
	GetUUID() string
	GetProject() string
}

type CommonProject interface {
	SendReport() error
	GetName() string
	GetMode() string
}
