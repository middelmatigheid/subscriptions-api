package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/middelmatigheid/subscriptions-api/internal/config"
	"github.com/middelmatigheid/subscriptions-api/internal/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// Creates cache
func NewCache(config *config.Config) (*Cache, error) {
	// Getting the redis client
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisHost + ":" + config.RedisPort,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Checking the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Cache{
		client: client,
		ttl:    time.Duration(config.RedisTTL) * time.Minute,
	}, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

// Get key to the subscription by its id
func (c *Cache) subID(id int) string {
	return fmt.Sprintf("sub:%d", id)
}

// Get key to the subscription by combination of user uuid and service name
func (c *Cache) subUserAndService(userUUID uuid.UUID, serviceName string) string {
	return fmt.Sprintf("sub:%s:%s", userUUID, serviceName)
}

// Cache in subscription
func (c *Cache) SetSubscription(ctx context.Context, subscription models.Subscription) error {
	data, err := json.Marshal(subscription)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.subID(subscription.ID), data, c.ttl).Err()
}

// Get the subscription from the cache
func (c *Cache) GetSubscription(ctx context.Context, identifier models.SubscriptionIdentifier) (*models.Subscription, error) {
	var data []byte
	var err error
	if identifier.ID > 0 {
		data, err = c.client.Get(ctx, c.subID(identifier.ID)).Bytes()
		if err == redis.Nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	} else {
		data, err = c.client.Get(ctx, c.subUserAndService(identifier.UserUUID, identifier.ServiceName)).Bytes()
		if err == redis.Nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}

	var subscription models.Subscription
	err = json.Unmarshal(data, &subscription)
	return &subscription, err
}

// Delete invalid subscription from the cache
func (c *Cache) DeleteSubscription(ctx context.Context, identifier models.SubscriptionIdentifier) error {
	err := c.client.Del(ctx, c.subID(identifier.ID)).Err()
	if err != nil {
		return err
	}
	return c.client.Del(ctx, c.subUserAndService(identifier.UserUUID, identifier.ServiceName)).Err()
}
