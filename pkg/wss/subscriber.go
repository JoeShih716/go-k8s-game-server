package wss

// Subscriber 定義了 WebSocket 事件的訂閱者介面。
// 任何實現此介面的類型都可以註冊到 Server，以接收連線、斷線和訊息事件。
type Subscriber interface {
	// OnConnect 當有新的客戶端連線建立時被呼叫。
	//
	// @param client 新建立的客戶端連線實例。
	OnConnect(client Client)

	// OnDisconnect 當一個客戶端連線中斷時被呼叫。
	//
	// @param client 已中斷的客戶端連線實例。
	OnDisconnect(client Client)

	// OnMessage 當從客戶端收到新的訊息時被呼叫。
	//
	// @param client 發送訊息的客戶端。
	// @param message 接收到的原始訊息內容。
	OnMessage(client Client, message []byte)
}
