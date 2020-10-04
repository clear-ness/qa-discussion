package clusters

import (
	"github.com/clear-ness/qa-discussion/model"
)

type ClusterMessageHandler interface {
	HandleClusterMessage(cm *model.ClusterMessage)
}

// appパッケージから呼ばれる。
func NewClusterMessageHandler(h func(*model.ClusterMessage)) clusterMessageHandler {
	return clusterMessageHandler{h}
}

type clusterMessageHandler struct {
	handlerFunc func(*model.ClusterMessage)
}

// wh.handlerFunc(r)で個別リクエストを処理する
func (h clusterMessageHandler) HandleClusterMessage(cm *model.ClusterMessage) {
	h.handlerFunc(cm)
}
