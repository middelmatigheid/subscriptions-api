package service

import (
	"context"
	"errors"

	"github.com/middelmatigheid/subscriptions-api/internal/cache"
	"github.com/middelmatigheid/subscriptions-api/internal/config"
	"github.com/middelmatigheid/subscriptions-api/internal/models"

	"github.com/google/uuid"
)

type Service struct {
	Database models.Storage
	Cache    *cache.Cache
}

func NewService(config *config.Config, db models.Storage) (*Service, error) {
	cache, err := cache.NewCache(config)
	if err != nil {
		return nil, nil
	}
	return &Service{Database: db, Cache: cache}, nil
}

// Validating subscription
func (s *Service) ValidateSubscription(subscription models.Subscription) error {
	// Validating user uuid
	if subscription.UserUUID == uuid.Nil {
		return models.NewErrBadRequest(errors.New("Empty user uuid"))
	}

	// Validating service name
	if len(subscription.ServiceName) == 0 {
		return models.NewErrBadRequest(errors.New("Empty service name"))
	}

	// Validating price
	if subscription.Price <= 0 {
		return models.NewErrBadRequest(errors.New("Invalid price"))
	}

	// Validating time bounds
	if !subscription.StartDate.Valid || (subscription.EndDate.Valid && subscription.EndDate.Time.Before(subscription.StartDate.Time)) {
		return models.NewErrBadRequest(errors.New("Invalid time bounds"))
	}
	return nil
}

// Creating new subscription
func (s *Service) Create(ctx context.Context, subscription models.Subscription) (models.IDResponse, error) {
	err := s.ValidateSubscription(subscription)
	if err != nil {
		return models.IDResponse{}, err
	}

	// Inserting the subscription into the database
	res, err := s.Database.Create(ctx, subscription)
	if s.Cache != nil {
		subscription.ID = res.ID
		s.Cache.SetSubscription(ctx, subscription)
	}
	return res, err
}

// Reading subscription
func (s *Service) Read(ctx context.Context, identifier models.SubscriptionIdentifier) (models.Subscription, error) {
	if identifier.ID == 0 && (identifier.UserUUID == uuid.Nil || len(identifier.ServiceName) == 0) {
		return models.Subscription{}, models.NewErrBadRequest(errors.New("Not enough arguments"))
	}

	if s.Cache != nil {
		sub, err := s.Cache.GetSubscription(ctx, identifier)
		if err == nil && sub != nil {
			return *sub, nil
		}
	}
	// Getting subscription's info from the database
	res, err := s.Database.Read(ctx, identifier)
	return res, err
}

// Updating the subscription
func (s *Service) Update(ctx context.Context, subscription models.Subscription) error {
	err := s.ValidateSubscription(subscription)
	if err != nil {
		return err
	}

	// Updating the subscription's info
	err = s.Database.Update(ctx, subscription)
	if s.Cache != nil {
		s.Cache.DeleteSubscription(ctx, models.SubscriptionIdentifier{ID: subscription.ID})
	}
	return err
}

// Updating the subscription partially
func (s *Service) Patch(ctx context.Context, subscriptionPatch models.SubscriptionPatch) error {
	exists, err := s.Database.Read(ctx, models.SubscriptionIdentifier{ID: subscriptionPatch.ID})
	if err != nil {
		return err
	}

	// Configuring updated subscription. If the field wasn't provided it remains unchanged
	var subscription models.Subscription
	subscription.ID = subscriptionPatch.ID

	// Getting user uuid
	if subscriptionPatch.UserUUID != nil {
		subscription.UserUUID = *subscriptionPatch.UserUUID
	} else {
		subscription.UserUUID = exists.UserUUID
	}

	// Getitng service name
	if subscriptionPatch.ServiceName != nil {
		subscription.ServiceName = *subscriptionPatch.ServiceName
	} else {
		subscription.ServiceName = exists.ServiceName
	}

	// Getting price
	if subscriptionPatch.Price != nil {
		subscription.Price = *subscriptionPatch.Price
	} else {
		subscription.Price = exists.Price
	}

	// Getting time bounds
	if subscriptionPatch.StartDate != nil {
		subscription.StartDate = *subscriptionPatch.StartDate
	} else {
		subscription.StartDate = exists.StartDate
	}
	if subscriptionPatch.EndDate != nil {
		subscription.EndDate = *subscriptionPatch.EndDate
	} else {
		subscription.EndDate = exists.EndDate
	}

	// Validating subscription
	err = s.ValidateSubscription(subscription)
	if err != nil {
		return err
	}

	// Updating the subscription's info
	err = s.Database.Update(ctx, subscription)
	if s.Cache != nil {
		s.Cache.DeleteSubscription(ctx, models.SubscriptionIdentifier{ID: subscription.ID})
	}
	return err
}

// Deleting the subscription
func (s *Service) Delete(ctx context.Context, identifier models.SubscriptionIdentifier) error {
	// Id or combination of user uuid and service name should be provided to specify the subscription
	if identifier.ID == 0 && (identifier.UserUUID == uuid.Nil || len(identifier.ServiceName) == 0) {
		return models.NewErrBadRequest(errors.New("Not enough arguments"))
	}

	// Deleting the subscription from the database
	err := s.Database.Delete(ctx, identifier)
	if s.Cache != nil {
		s.Cache.DeleteSubscription(ctx, identifier)
	}
	return err
}

// Gettng list of subscrtiption
func (s *Service) List(ctx context.Context, params models.SubscriptionsWithinPeriod) ([]models.Subscription, error) {
	// Validating time bounds
	if params.EndDate.Valid && params.StartDate.Valid && params.EndDate.Time.Before(params.StartDate.Time) {
		return []models.Subscription{}, models.NewErrBadRequest(errors.New("Invalid time bound"))
	}
	// Getting list of subscriptions from the database
	res, err := s.Database.List(ctx, params)
	return res, err
}

// Getting summary of subscriptions
func (s *Service) Summary(ctx context.Context, params models.SubscriptionsWithinPeriod) (models.SummaryResponse, error) {
	// Validating time bounds
	if params.EndDate.Valid && params.StartDate.Valid && params.EndDate.Time.Before(params.StartDate.Time) {
		return models.SummaryResponse{}, models.NewErrBadRequest(errors.New("Invalid time bound"))
	}
	// Getting info from the database
	res, err := s.Database.Summary(ctx, params)
	return res, err
}
