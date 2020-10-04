package model

import (
	"encoding/json"
	"io"
)

const (
	// webSocketメッセージをクラスタ全体に周知する場合。
	CLUSTER_EVENT_WEBSOCKET = "websocket"

	// L1キャッシュの削除をクラスタ全体に周知する場合。
	CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_USER = "clear_user_session"
)

type ClusterMessage struct {
	// 自サーバーのインスタンスID
	// を指定する事で自分以外のサーバー達だけに処理させる
	OmitCluster string `json:"omit_cluster"`
	// websocket, clear_user_session のいずれか
	Event string `json:"event"`
	// WebSocketEventのjson形式 または user_id
	Data string `json:"data,omitempty"`
	// 必要なら使う
	Props map[string]string `json:"props,omitempty"`
}

func (o *ClusterMessage) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func ClusterMessageFromJson(data io.Reader) *ClusterMessage {
	var o *ClusterMessage
	json.NewDecoder(data).Decode(&o)
	return o
}
