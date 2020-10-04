package clusters

import (
	"context"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/configservice"

	"github.com/go-redis/redis/v8"
)

type ClusterInterface interface {
	Start(clusterId string)
	RegisterClusterMessageHandler(event string, cmh ClusterMessageHandler)
	SendClusterMessage(cm *model.ClusterMessage)
}

type ClusterImpl struct {
	configService configservice.ConfigService
	handlers      map[string]ClusterMessageHandler
}

func MakeCluster(configService configservice.ConfigService) ClusterInterface {
	return &ClusterImpl{
		configService,
		make(map[string]ClusterMessageHandler),
	}
}

// appパッケージから呼ばれる。
func (h *ClusterImpl) RegisterClusterMessageHandler(event string, cmh ClusterMessageHandler) {
	h.handlers[event] = cmh
}

func GetAllClusterChannels() []string {
	allChannels := []string{model.CLUSTER_EVENT_WEBSOCKET, model.CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_USER}
	return allChannels
}

func (h *ClusterImpl) ClusterClient() *redis.Client {
	addr := *h.configService.Config().ClusterSettings.ClusterEndpoint

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return client
}

func (h *ClusterImpl) ServeClusterMessage(cm *model.ClusterMessage) {
	// 各eventに応じたstructを選ぶ
	handler, ok := h.handlers[cm.Event]
	if !ok {
		return
	}

	// 個別リクエストとして処理する
	handler.HandleClusterMessage(cm)
}

func (h *ClusterImpl) Start(clusterId string) {
	var ctx = context.Background()
	go func() {
		h.subscribe(ctx, clusterId)
	}()
}

func (h *ClusterImpl) subscribe(ctx context.Context, clusterId string) {
	client := h.ClusterClient()

	channels := GetAllClusterChannels()

	// There is no error because go-redis automatically reconnects on error.
	pubsub := client.Subscribe(ctx, channels...)
	defer pubsub.Unsubscribe(ctx)

	ch := pubsub.Channel()
	for msg := range ch {
		cm := model.ClusterMessageFromJson(strings.NewReader(msg.Payload))
		// 自サーバーの場合は無視
		if cm.OmitCluster == clusterId {
			continue
		}

		h.ServeClusterMessage(cm)
	}
}

func (h *ClusterImpl) SendClusterMessage(cm *model.ClusterMessage) {
	var ctx = context.Background()
	go func() {
		h.publish(ctx, cm)
	}()
}

func (h *ClusterImpl) publish(ctx context.Context, cm *model.ClusterMessage) {
	client := h.ClusterClient()

	// ClusterMessage.Eventをそのままredis pub/subのチャンネル名にする
	err := client.Publish(ctx, cm.Event, cm.ToJson()).Err()
	if err != nil {
		panic(err)
	}
}
