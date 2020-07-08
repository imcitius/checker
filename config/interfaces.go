package config

//type ChatAlert interface {
//	SendAlert(alerttype string, e error) error
//	GetAlertName() string
//	GetAlertType() string
//	GetAlertCreds() string
//}

type IncomingChatMessage interface {
	GetUUID() (string, error)
	GetProject() (string, error)
}

type CommonProject interface {
	SendReport() error
	GetName() string
	GetMode() string
	Loud()
	Quiet()
	Send()
	SendCrit()
}
