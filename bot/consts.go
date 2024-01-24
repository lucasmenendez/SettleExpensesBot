package bot

const (
	updatesEndpointTemplate = "https://api.telegram.org/bot%s/getUpdates?offset=%d"
	baseEndpointTemplate    = "https://api.telegram.org/bot%s/%s"
)

const (
	sendMessageMethod            = "sendMessage"
	editMessageTextMethod        = "editMessageText"
	editMessageReplyMarkupMethod = "editMessageReplyMarkup"
	removeMessageMethod          = "deleteMessage"
	sendDocumentMethod           = "sendDocument"
	getFileMethod                = "getFile"
)
