package wss

// clientMessage 是一個內部結構，用於將客戶端和其發送的訊息綁定在一起。
type clientMessage struct {
	client  *connection
	message []byte
}
