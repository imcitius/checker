package alerts

//func GetAlerters() (TAlertersCollection, error) {
//	alerters := TAlertersCollection{
//		Alerters: make(map[string]ICommonAlerter),
//	}
//
//	if len(alertsConfig.Alerts) > 0 {
//		if alertsConfig.Alerts[0].BotToken != "" {
//			alerter := &tg.TTelegramAlerter{
//				Token: alertsConfig.Alerts[0].BotToken,
//				Log:   Log,
//			}
//			//alerter.Init()
//			//alerter.Send(1390752, "fsdfsdfsdfsdfsdfd")
//			alerters.Alerters[alertsConfig.Alerts[0].Name] = alerter
//		}
//	}
//
//	alerters.Alerters["log"] = &log.TLogAlerter{
//		Log: Log,
//	}
//
//	return alerters, nil
//}
