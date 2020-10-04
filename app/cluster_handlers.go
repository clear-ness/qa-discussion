package app

import (
	"strings"

	"github.com/clear-ness/qa-discussion/clusters"
	"github.com/clear-ness/qa-discussion/model"
)

// redis pub/sub経由でClusterMessageを受け取った際のハンドリング処理を登録
func (a *App) registerAllClusterMessageHandlers() {
	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_WEBSOCKET, clusters.NewClusterMessageHandler(a.clusterPublishHandler))

	a.Cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_USER, clusters.NewClusterMessageHandler(a.clusterClearSessionCacheForUserHandler))
}

func (a *App) clusterPublishHandler(msg *model.ClusterMessage) {
	event := model.WebSocketEventFromJson(strings.NewReader(msg.Data))
	if event == nil {
		return
	}

	a.LocalPublish(event)
}

func (a *App) clusterClearSessionCacheForUserHandler(msg *model.ClusterMessage) {
	a.ClearLocalSessionCacheForUser(msg.Data)
}
