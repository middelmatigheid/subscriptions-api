package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/middelmatigheid/subscriptions-api/internal/config"
	"github.com/middelmatigheid/subscriptions-api/internal/models"
	"github.com/middelmatigheid/subscriptions-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	Service models.SubscriptionService
}

func NewHandler(config *config.Config, db models.Storage) (*Handler, error) {
	service, err := service.NewService(config, db)
	if err != nil {
		return nil, nil
	}
	return &Handler{Service: service}, nil
}

// @Summary Create a new subscription
// @Description The endpoint inserts a new subscription to the database. If another subscription with the same user uuid and service name already exists in the database a conflict error will be thrown
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.Subscription true "Subscription data"
// @Success 201 {object} models.IDResponse
// @Success 409
// @Failure 400
// @Failure 500
// @Router /create [post]
func (h *Handler) Create(c *gin.Context) {
	// Reading request's body
	var subscription models.Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Error while reading request's body", "error": err.Error()})
		return
	}

	// Inserting the subscription into the database
	ctx := c.Request.Context()
	res, err := h.Service.Create(ctx, subscription)
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Internal server error", "error": err.Error()})
		return
	case errors.Is(err, models.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"msg": "The subscription is already being stored in the database", "error": err.Error(), "body": res})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusCreated, gin.H{"msg": "The subscription successfully created", "body": res})
}

// @Summary Get subscription information
// @Description The endpoints return subscription's info. The subscription is being specified by its id or combination of user uuid and service name
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id query int false "1"
// @Param user_uuid query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Success 200 {object} models.Subscription
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /read [get]
func (h *Handler) Read(c *gin.Context) {
	// Getting query params
	id, err := strconv.Atoi(c.DefaultQuery("id", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid id", "error": err.Error()})
		return
	}
	userUUID, err := uuid.Parse(c.DefaultQuery("user_uuid", "00000000-0000-0000-0000-000000000000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user uuid", "error": err.Error()})
		return
	}
	serviceName := c.DefaultQuery("service_name", "")

	// Getting subscription's info from the database
	ctx := c.Request.Context()
	res, err := h.Service.Read(ctx, models.SubscriptionIdentifier{ID: id, UserUUID: userUUID, ServiceName: serviceName})
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscription info from the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscrtiption was successfully read", "body": res})
}

// @Summary Update subscription
// @Description The endpoint updates existing subscription's info. The subscription is being specified by its id. All fields should be provided. If another subscription with the same user uuid and service name already exists in the database a conflict error will be thrown
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.Subscription true "Updated subscription data"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /update [put]
func (h *Handler) Update(c *gin.Context) {
	// Readind request's body
	var subscription models.Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Error while reading request's body", "error": err.Error()})
		return
	}

	// Updating the subscription's info
	ctx := c.Request.Context()
	err := h.Service.Update(ctx, subscription)
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while updating subscription info from the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"msg": "The subscription is already being stored in the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscription was successfully updated"})
}

// @Summary Partial subscription update
// @Description The endpoints updates existing subscription's info partially. The subscription is being specified by its id. If another subscription with the same user uuid and service name already exists in the database a conflict error will be thrown. Only updating fields can be specified, other fields will remain the same
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.SubscriptionPatch true "Updated subscription data"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /patch [put]
func (h *Handler) Patch(c *gin.Context) {
	// Readind request's body
	var subscriptionPatch models.SubscriptionPatch
	if err := c.ShouldBindJSON(&subscriptionPatch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Error while reading request's body", "error": err.Error()})
		return
	}

	// Updating the subscription's info
	ctx := c.Request.Context()
	err := h.Service.Patch(ctx, subscriptionPatch)
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while updating subscription info from the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscription was successfully updated"})
}

// @Summary Delete subscription
// @Description The endpoint deletes subscription from the database. The subscription is being specified by its id or combination of user uuid and service name
// @Tags subscriptions
// @Produce json
// @Param id query int false "1"
// @Param user_uuid query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Success 200
// @Failure 400
// @Failure 500
// @Router /delete [delete]
func (h *Handler) Delete(c *gin.Context) {
	// Getting query params
	id, err := strconv.Atoi(c.DefaultQuery("id", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid id", "error": err.Error()})
		return
	}
	userUUID, err := uuid.Parse(c.DefaultQuery("user_uuid", "00000000-0000-0000-0000-000000000000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user uuid", "error": err.Error()})
		return
	}
	serviceName := c.DefaultQuery("service_name", "")

	// Deleting the subscription from the database
	ctx := c.Request.Context()
	err = h.Service.Delete(ctx, models.SubscriptionIdentifier{ID: id, UserUUID: userUUID, ServiceName: serviceName})
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while deleting subscription info from the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscription was successfully deleted"})
}

// @Summary Get list of subscriptions
// @Description The endpoint gets list of subscriptions. The list can be filtered by user uuid, service name, start date and end date
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_uuid query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Param start_date query string false "07-2025"
// @Param end_date query string false "08-2025"
// @Param limit query int false "10"
// @Param offset query int false "0"
// @Success 200 {array} models.Subscription
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /list [get]
func (h *Handler) List(c *gin.Context) {
	// Getting query params
	userUUID, err := uuid.Parse(c.DefaultQuery("user_uuid", "00000000-0000-0000-0000-000000000000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user uuid", "error": err.Error()})
		return
	}
	serviceName := c.DefaultQuery("service_name", "")
	// Getting start date
	start := c.DefaultQuery("start_date", "")
	var startDate models.CustomDate
	if len(start) > 0 {
		date, err := time.Parse("01-2006", start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error": err.Error()})
			return
		}
		startDate = models.CustomDate{NullTime: sql.NullTime{Time: date, Valid: true}}
	} else {
		startDate.Valid = false
	}
	// Getting end date
	end := c.DefaultQuery("end_date", "")
	var endDate models.CustomDate
	if len(end) > 0 {
		date, err := time.Parse("01-2006", end)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error": err.Error()})
			return
		}
		endDate = models.CustomDate{NullTime: sql.NullTime{Time: date, Valid: true}}
	} else {
		endDate.Valid = false
	}

	// Getting limit
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid limit", "error": err.Error()})
		return
	}
	// Getting offset
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid offset", "error": err.Error()})
		return
	}

	// Getting list of subscriptions from the database
	ctx := c.Request.Context()
	res, err := h.Service.List(ctx, models.SubscriptionsWithinPeriod{UserUUID: userUUID, ServiceName: serviceName, StartDate: startDate, EndDate: endDate, Limit: limit, Offset: offset})
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscriptions info from the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subcriptions were successfully read", "body": res})
}

// @Summary Get total sum of subscriptions prices
// @Description The endpoints returns total amount of unique subscriptions and calculates its total price within the provided period. It is implied that both of start date and end date is being paid. The subscriptions can be filtered by user id or service name
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_uuid query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Param start_date query string true "07-2025"
// @Param end_date query string true "08-2025"
// @Success 200 {object} models.SummaryResponse
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /summary [get]
func (h *Handler) Summary(c *gin.Context) {
	// Getting query params
	userUUID, err := uuid.Parse(c.DefaultQuery("user_uuid", "00000000-0000-0000-0000-000000000000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user uuid", "error": err.Error()})
		return
	}
	serviceName := c.DefaultQuery("service_name", "")
	// Validating start date
	start := c.DefaultQuery("start_date", "")
	var startDate models.CustomDate
	if len(start) > 0 {
		date, err := time.Parse("01-2006", start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error": err.Error()})
			return
		}
		startDate = models.CustomDate{NullTime: sql.NullTime{Time: date, Valid: true}}
	} else {
		startDate.Valid = false
	}
	// Validating end date
	end := c.DefaultQuery("end_date", "")
	var endDate models.CustomDate
	if len(end) > 0 {
		date, err := time.Parse("01-2006", end)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error": err.Error()})
			return
		}
		endDate = models.CustomDate{NullTime: sql.NullTime{Time: date, Valid: true}}
	} else {
		endDate.Valid = false
	}

	// Getting info from the database
	ctx := c.Request.Context()
	res, err := h.Service.Summary(ctx, models.SubscriptionsWithinPeriod{UserUUID: userUUID, ServiceName: serviceName, StartDate: startDate, EndDate: endDate})
	switch {
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request", "error": err.Error()})
		return
	case errors.Is(err, models.ErrInternalServer):
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscriptions info from the database", "error": err.Error()})
		return
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscriptions are not found in the database", "error": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error": err.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The total sum was successfully calculated", "body": res})
}
