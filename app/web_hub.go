package app

import (
	"hash/maphash"
	"runtime"
	"sync/atomic"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
)

const (
	broadcastQueueSize = 4096
)

type webConnDirectMessage struct {
	conn *WebConn
	msg  model.WebSocketMessage
}

type Hub struct {
	// connectionCount should be kept first.
	connectionCount int64
	app             *App
	register        chan *WebConn
	unregister      chan *WebConn
	broadcast       chan *model.WebSocketEvent
	stop            chan struct{}
	didStop         chan struct{}
	invalidateUser  chan string
	directMsg       chan *webConnDirectMessage
	explicitStop    bool
}

// NewWebHub creates a new Hub.
func (a *App) NewWebHub() *Hub {
	return &Hub{
		app:            a,
		register:       make(chan *WebConn),
		unregister:     make(chan *WebConn),
		broadcast:      make(chan *model.WebSocketEvent, broadcastQueueSize),
		stop:           make(chan struct{}),
		didStop:        make(chan struct{}),
		invalidateUser: make(chan string),
		directMsg:      make(chan *webConnDirectMessage),
	}
}

// 1 サーバーが複数Hubを持つ設計ならこれが必要。
// GetHubForUserId returns the hub for a given user id.
func (s *Server) GetHubForUserId(userId string) *Hub {
	var hash maphash.Hash
	hash.SetSeed(s.hashSeed)
	hash.Write([]byte(userId))

	index := hash.Sum64() % uint64(len(s.hubs))

	return s.hubs[int(index)]
}

func (a *App) GetHubForUserId(userId string) *Hub {
	return a.Srv.GetHubForUserId(userId)
}

func (a *App) HubStart() {
	numberOfHubs := runtime.NumCPU() * 2
	hubs := make([]*Hub, numberOfHubs)

	for i := 0; i < numberOfHubs; i++ {
		hubs[i] = a.NewWebHub()
		hubs[i].Start()
	}

	// Assigning to the hubs slice without any mutex is fine because it is only assigned once
	// during the start of the program and always read from after that.
	a.Srv.hubs = hubs
}

func (s *Server) HubStop() {
	for _, hub := range s.hubs {
		hub.Stop()
	}
}

func (h *Hub) Stop() {
	close(h.stop)
	<-h.didStop
}

// (現サーバーにwebSocket接続中の)特定ユーザーのwebConn達が持つユーザーセッション情報を初期化する
func (a *App) InvalidateWebConnSessionCacheForUser(userId string) {
	hub := a.GetHubForUserId(userId)
	if hub != nil {
		hub.InvalidateUser(userId)
	}
}

func (h *Hub) InvalidateUser(userId string) {
	select {
	case h.invalidateUser <- userId:
	case <-h.stop:
	}
}

// sends the given message to the given connection.
func (h *Hub) SendMessage(conn *WebConn, msg model.WebSocketMessage) {
	select {
	case h.directMsg <- &webConnDirectMessage{
		conn: conn,
		msg:  msg,
	}:
	case <-h.stop:
	}
}

type hubConnectionIndex struct {
	// ユーザーIDからwebConn(複数)を取得できる様にしている
	byUserId map[string][]*WebConn
	// すべてのwebConnについてそれぞれ
	// byUserIdのWebConnリスト中のどのindex値かをwebConnから引ける様にしている
	byConnection map[*WebConn]int
}

func newHubConnectionIndex() *hubConnectionIndex {
	return &hubConnectionIndex{
		byUserId:     make(map[string][]*WebConn),
		byConnection: make(map[*WebConn]int),
	}
}

func (i *hubConnectionIndex) Add(wc *WebConn) {
	i.byUserId[wc.UserId] = append(i.byUserId[wc.UserId], wc)
	// 追記していく
	i.byConnection[wc] = len(i.byUserId[wc.UserId]) - 1
}

func (i *hubConnectionIndex) Remove(wc *WebConn) {
	userConnIndex, ok := i.byConnection[wc]
	if !ok {
		return
	}

	userConnections := i.byUserId[wc.UserId]
	last := userConnections[len(userConnections)-1]
	// set the slot that we are trying to remove to be the last connection.
	userConnections[userConnIndex] = last
	// remove the last connection from the slice.
	i.byUserId[wc.UserId] = userConnections[:len(userConnections)-1]
	// set the index of the connection that was moved to the new index.
	i.byConnection[last] = userConnIndex

	delete(i.byConnection, wc)
}

func (i *hubConnectionIndex) Has(wc *WebConn) bool {
	_, ok := i.byConnection[wc]
	return ok
}

func (i *hubConnectionIndex) ForUser(id string) []*WebConn {
	return i.byUserId[id]
}

func (i *hubConnectionIndex) All() map[*WebConn]int {
	return i.byConnection
}

func (h *Hub) Start() {
	var doStart func()
	var doRecoverableStart func()
	var doRecover func()

	doStart = func() {
		connIndex := newHubConnectionIndex()

		// (比較的少人数な)inboxMessageのwebSocket通知、がまずはメイン。
		// (別のサーバーにストリームアップが来てredis Pubされた後の)
		// redis Subメッセージ受信 → webHubへの対応は、
		// Pub/Subメッセージ内のuserIdが現サーバーのwebHubが保持するwebConnリストの
		// いずれかに一致しているかを確認し、一致があれば以降の実装はmattと同じ。
		for {
			select {
			// http (upgrade) APIでconnectWebSocketを呼ばれた際に受信する
			case webConn := <-h.register:
				connIndex.Add(webConn)

				atomic.StoreInt64(&h.connectionCount, int64(len(connIndex.All())))

				if webConn.IsAuthenticated() {
					webConn.send <- webConn.createHelloMessage()
				}
			// 1 webConがPumpをやめた際に呼ばれる
			case webConn := <-h.unregister:
				connIndex.Remove(webConn)
				atomic.StoreInt64(&h.connectionCount, int64(len(connIndex.All())))

			case userId := <-h.invalidateUser:
				for _, webConn := range connIndex.ForUser(userId) {
					webConn.InvalidateCache()
				}
			case directMsg := <-h.directMsg:
				if !connIndex.Has(directMsg.conn) {
					continue
				}

				select {
				case directMsg.conn.send <- directMsg.msg:
					// 送信に成功した場合
				default:
					// 送信に失敗した場合
					close(directMsg.conn.send)
					connIndex.Remove(directMsg.conn)
				}
			case msg := <-h.broadcast:
				msg = msg.PrecomputeJSON()

				broadcast := func(webConn *WebConn) {
					if !connIndex.Has(webConn) {
						return
					}

					if webConn.shouldSendEvent(msg) {
						select {
						// 後にapp/web_connのwritePumpメソッドで受信される
						case webConn.send <- msg:
							// 送信に成功した場合
						default:
							// 送信に失敗した場合
							close(webConn.send)
							connIndex.Remove(webConn)
						}
					}
				}

				if msg.GetBroadcast().UserId != "" {
					candidates := connIndex.ForUser(msg.GetBroadcast().UserId)
					for _, webConn := range candidates {
						broadcast(webConn)
					}

					continue
				}

				candidates := connIndex.All()
				for webConn := range candidates {
					broadcast(webConn)
				}
			// サーバーを停止する場合に呼ばれる
			case <-h.stop:
				for webConn := range connIndex.All() {
					webConn.Close()
				}

				h.explicitStop = true
				close(h.didStop)

				return
			}
		}
	}

	doRecoverableStart = func() {
		defer doRecover()
		doStart()
	}

	doRecover = func() {
		if !h.explicitStop {
			if r := recover(); r != nil {
				mlog.Error("hub panic", mlog.Any("panic", r))
			} else {
				mlog.Error("hub panic")
			}

			go doRecoverableStart()
		}
	}

	go doRecoverableStart()
}

func (a *App) LocalPublish(message *model.WebSocketEvent) {
	a.Srv.LocalPublish(message)
}

// 現サーバーのHubに接続中のsocket達にWebSocketEventメッセージを送信する
func (s *Server) LocalPublish(message *model.WebSocketEvent) {
	if message.GetBroadcast().UserId != "" {
		hub := s.GetHubForUserId(message.GetBroadcast().UserId)

		if hub != nil {
			hub.Broadcast(message)
		}
	} else {
		for _, hub := range s.hubs {
			hub.Broadcast(message)
		}
	}
}

// TODO: inboxMessage作成時にwebSocketで通知する様に
// 現サーバーのHubに接続中のsocket達にWebSocketEventを送信後、
// (自サーバーを除く)クラスター全体にも周知する
func (s *Server) Publish(message *model.WebSocketEvent) {
	s.LocalPublish(message)

	if s.Cluster != nil {
		cm := &model.ClusterMessage{
			OmitCluster: s.clusterId,
			Event:       model.CLUSTER_EVENT_WEBSOCKET,
			Data:        message.ToJson(),
		}

		s.Cluster.SendClusterMessage(cm)
	}
}

// broadcasts the message to all connections in the hub.
func (h *Hub) Broadcast(message *model.WebSocketEvent) {
	if h != nil && message != nil {
		select {
		// 後に各サーバーは
		// このファイル内のStart()メソッド内で受信する
		case h.broadcast <- message:
		case <-h.stop:
		}
	}
}

func (a *App) HubRegister(webConn *WebConn) {
	hub := a.GetHubForUserId(webConn.UserId)
	if hub != nil {
		hub.Register(webConn)
	}
}

func (h *Hub) Register(webConn *WebConn) {
	select {
	case h.register <- webConn:
	case <-h.stop:
	}
}

func (a *App) HubUnregister(webConn *WebConn) {
	hub := a.GetHubForUserId(webConn.UserId)
	if hub != nil {
		hub.Unregister(webConn)
	}
}

func (h *Hub) Unregister(webConn *WebConn) {
	select {
	case h.unregister <- webConn:
	case <-h.stop:
	}
}
