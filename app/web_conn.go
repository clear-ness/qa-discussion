package app

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/gorilla/websocket"
)

const (
	sendQueueSize = 256

	pongWaitTime      = 100 * time.Second
	authCheckInterval = 5 * time.Second
	writeWaitTime     = 30 * time.Second
	sendSlowWarn      = (sendQueueSize * 50) / 100
)

// a single websocket connection to a user.
type WebConn struct {
	sessionExpiresAt int64 // This should stay at the top for 64-bit alignment of 64-bit words accessed atomically
	App              *App
	WebSocket        *websocket.Conn
	Sequence         int64
	UserId           string
	send             chan model.WebSocketMessage
	sessionToken     atomic.Value
	session          atomic.Value
	endWritePump     chan struct{}
	pumpFinished     chan struct{}
}

func (a *App) NewWebConn(ws *websocket.Conn, session model.Session) *WebConn {
	if session.UserId != "" {
		a.Srv.Go(func() {
			a.SetStatusOnline(session.UserId, false)
		})
	}

	wc := &WebConn{
		App:          a,
		send:         make(chan model.WebSocketMessage, sendQueueSize),
		WebSocket:    ws,
		UserId:       session.UserId,
		endWritePump: make(chan struct{}),
		pumpFinished: make(chan struct{}),
	}

	wc.SetSession(&session)
	wc.SetSessionToken(session.Token)
	wc.SetSessionExpiresAt(session.ExpiresAt)

	return wc
}

// GetSessionExpiresAt returns the time at which the session expires.
func (wc *WebConn) GetSessionExpiresAt() int64 {
	return atomic.LoadInt64(&wc.sessionExpiresAt)
}

// SetSessionExpiresAt sets the time at which the session expires.
func (wc *WebConn) SetSessionExpiresAt(v int64) {
	atomic.StoreInt64(&wc.sessionExpiresAt, v)
}

// GetSessionToken returns the session token of the connection.
func (wc *WebConn) GetSessionToken() string {
	return wc.sessionToken.Load().(string)
}

// SetSessionToken sets the session token of the connection.
func (wc *WebConn) SetSessionToken(v string) {
	wc.sessionToken.Store(v)
}

// GetSession returns the session of the connection.
func (wc *WebConn) GetSession() *model.Session {
	return wc.session.Load().(*model.Session)
}

// SetSession sets the session of the connection.
func (wc *WebConn) SetSession(v *model.Session) {
	if v != nil {
		v = v.DeepCopy()
	}

	wc.session.Store(v)
}

func (wc *WebConn) IsAuthenticated() bool {
	// Check the expiry to see if we need to check for a new session
	if wc.GetSessionExpiresAt() < model.GetMillis() {
		if wc.GetSessionToken() == "" {
			return false
		}

		// L1キャッシュ→DB の順にユーザーセッションを確認
		session, err := wc.App.GetSession(wc.GetSessionToken())
		if err != nil {
			wc.SetSessionToken("")
			wc.SetSession(nil)
			wc.SetSessionExpiresAt(0)
			return false
		}

		// 取得したユーザーセッションをwebConnに紐付ける
		wc.SetSession(session)
		wc.SetSessionExpiresAt(session.ExpiresAt)
	}

	return true
}

// starts the WebConn instance. After this, the websocket
// is ready to send/receive messages.
func (wc *WebConn) Pump() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		wc.writePump()
	}()

	wc.readPump()
	close(wc.endWritePump)

	wg.Wait()

	wc.App.HubUnregister(wc)
	close(wc.pumpFinished)
}

func (wc *WebConn) readPump() {
	defer func() {
		wc.WebSocket.Close()
	}()

	wc.WebSocket.SetReadLimit(model.SOCKET_MAX_MESSAGE_SIZE_KB)
	wc.WebSocket.SetReadDeadline(time.Now().Add(pongWaitTime))

	for {
		var req model.WebSocketRequest

		// 現サーバーにwebSocket接続中のエンドユーザーからのデータを読む
		if err := wc.WebSocket.ReadJSON(&req); err != nil {
			return
		}

		wc.App.Srv.WebSocketRouter.ServeWebSocket(wc, &req)
	}
}

func (wc *WebConn) writePump() {
	authTicker := time.NewTicker(authCheckInterval)

	defer func() {
		authTicker.Stop()
		wc.WebSocket.Close()
	}()

	for {
		select {
		// webHubにてshouldSendEvent確認の後にメッセージが送られてくる
		// webHubにregister後にhelloMessageが来る
		case msg, ok := <-wc.send:
			if !ok {
				wc.WebSocket.SetWriteDeadline(time.Now().Add(writeWaitTime))
				wc.WebSocket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			evt, evtOk := msg.(*model.WebSocketEvent)

			skipSend := false
			if len(wc.send) >= sendSlowWarn {
				// When the pump starts to get slow we'll drop non-critical messages
				switch msg.EventType() {
				case model.WEBSOCKET_EVENT_INBOX_MESSAGE:
					skipSend = true
				}
			}

			if skipSend {
				continue
			}

			var msgBytes []byte
			if evtOk {
				// evtのコピーにSequenceを更新したものを取得
				cpyEvt := evt.SetSequence(wc.Sequence)
				msgBytes = []byte(cpyEvt.ToJson())
				wc.Sequence++
			} else {
				msgBytes = []byte(msg.ToJson())
			}

			wc.WebSocket.SetWriteDeadline(time.Now().Add(writeWaitTime))
			// webConnの保持するソケットに対してテキストメッセージを書き込む
			if err := wc.WebSocket.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				return
			}

		case <-wc.endWritePump:
			return

		case <-authTicker.C:
			// webConn内のsessionTokenの存在を5秒おきに確認
			if wc.GetSessionToken() == "" {
				return
			}

			authTicker.Stop()
		}
	}
}

func (wc *WebConn) createHelloMessage() *model.WebSocketEvent {
	msg := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_HELLO, "", wc.UserId, nil)
	return msg
}

func (wc *WebConn) InvalidateCache() {
	wc.SetSession(nil)
	wc.SetSessionExpiresAt(0)
}

func (wc *WebConn) Close() {
	wc.WebSocket.Close()
	<-wc.pumpFinished
}

func (wc *WebConn) shouldSendEvent(msg *model.WebSocketEvent) bool {
	if !wc.IsAuthenticated() {
		return false
	}

	// 特定ユーザー向けのメッセージかどうかを確認
	if msg.GetBroadcast().UserId != "" {
		return wc.UserId == msg.GetBroadcast().UserId
	}

	// 省略指定も可能に
	if len(msg.GetBroadcast().OmitUsers) > 0 {
		if _, ok := msg.GetBroadcast().OmitUsers[wc.UserId]; ok {
			return false
		}
	}

	// Only report events to users who are in the team for the event
	if msg.GetBroadcast().TeamId != "" {
		return wc.isMemberOfTeam(msg.GetBroadcast().TeamId)
	}

	return true
}

func (wc *WebConn) isMemberOfTeam(teamId string) bool {
	currentSession := wc.GetSession()

	if currentSession == nil || currentSession.Token == "" {
		session, err := wc.App.GetSession(wc.GetSessionToken())
		if err != nil {
			return false
		}

		wc.SetSession(session)

		currentSession = session
	}

	return currentSession.GetTeamByTeamId(teamId) != nil
}
