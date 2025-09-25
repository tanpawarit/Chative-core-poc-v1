package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"
	errx "github.com/Chative-core-poc-v1/server/internal/core/error"
	logx "github.com/Chative-core-poc-v1/server/pkg/logger"
	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

type RedisConversationRepository struct {
	rdb redis.Cmdable
	ttl time.Duration
}

func NewRedisConversationRepository(rdb redis.Cmdable, ttl time.Duration) *RedisConversationRepository {
	return &RedisConversationRepository{rdb: rdb, ttl: ttl}
}

func (r *RedisConversationRepository) conversationKey(conversationID string) string {
	return fmt.Sprintf("conversation:%s:messages", conversationID)
}

func (r *RedisConversationRepository) AddMessage(ctx context.Context, conversationID string, message *schema.Message) error {
	b, err := json.Marshal(message)
	if err != nil {
		logx.Error().Err(err).Str("conversationID", conversationID).Msg("failed to marshal message")
		return fmt.Errorf("marshal message: %w", err)
	}
	key := r.conversationKey(conversationID)

	// append message
	if err := r.rdb.RPush(ctx, key, b).Err(); err != nil {
		logx.Error().Err(err).Str("key", key).Msg("failed to push message to redis")
		return errx.WrapRedis(err)
	}
	// extend TTL on touch
	if r.ttl > 0 {
		if ok, err := r.rdb.Expire(ctx, key, r.ttl).Result(); err != nil {
			logx.Error().Err(err).Str("key", key).Msg("failed to set expire")
			return errx.WrapRedis(err)
		} else if !ok {
			logx.Warn().Str("key", key).Dur("ttl", r.ttl).Msg("failed to set TTL on conversation key")
		}
	}
	return nil
}

func (r *RedisConversationRepository) LoadHistory(ctx context.Context, conversationID string) (*model.ConversationHistory, error) {
	key := r.conversationKey(conversationID)

	rows, err := r.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return &model.ConversationHistory{ConversationID: conversationID, Messages: []*schema.Message{}}, nil
		}
		logx.Error().Err(err).Str("key", key).Msg("failed to load conversation history from redis")
		return nil, errx.WrapRedis(err)
	}

	msgs := make([]*schema.Message, 0, len(rows))
	for i, s := range rows {
		var m schema.Message
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			logx.Error().Err(err).Str("conversationID", conversationID).Int("index", i).Msg("failed to unmarshal message")
			return nil, fmt.Errorf("unmarshal message at index %d: %w", i, err)
		}
		msgs = append(msgs, &m)
	}
	return &model.ConversationHistory{ConversationID: conversationID, Messages: msgs}, nil
}

func (r *RedisConversationRepository) ClearHistory(ctx context.Context, conversationID string) error {
	key := r.conversationKey(conversationID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logx.Error().Err(err).Str("key", key).Msg("failed to delete conversation history from redis")
		return errx.WrapRedis(err)
	}
	return nil
}

func (r *RedisConversationRepository) GetMessageCount(ctx context.Context, conversationID string) (int, error) {
	key := r.conversationKey(conversationID)
	n, err := r.rdb.LLen(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		logx.Error().Err(err).Str("key", key).Msg("failed to get message count from redis")
		return 0, errx.WrapRedis(err)
	}
	return int(n), nil
}

var _ model.ConversationRepository = (*RedisConversationRepository)(nil)
