package bot

const (
	updatesEndpointTemplate = "https://api.telegram.org/bot%s/getUpdates?offset=%d"
	baseEndpointTemplate    = "https://api.telegram.org/bot%s/%s"
)

const (
	sendMessageMethod            = "sendMessage"
	editMessageReplyMarkupMethod = "editMessageReplyMarkup"
	removeMessageMethod          = "deleteMessage"
)
