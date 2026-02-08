package handlers

import (
	"middelmatigheid/internal/database"
	"net/http"

	. "middelmatigheid/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	Db *database.Database
}

// @Summary Create a new subscription
// @Description Inserts a new subscription to the database
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.Subscription true "Subscription data"
// @Success 201 {object} models.IdResponse
// @Success 409
// @Failure 400
// @Failure 500
// @Router /create [post]
func (h *Handler) Create(c *gin.Context) {
	// Reading request's body
	var subscription Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Error while reading request's body", "error": err.Error()})
		return
	}

	// Validating service name
	if len(subscription.ServiceName) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Empty service name"})
		return
	}
	// Validating user id
	if _, err := uuid.Parse(subscription.UserId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id", "error": err.Error()})
		return
	}
	// Validating price
	if subscription.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid price"})
		return
	}

	// Inserting the subscription into the database
	res, customErr := h.Db.Create(subscription)
	if customErr != nil && customErr.TypeOf == ErrTimeParse {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Invalid price", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrConflict {
		c.JSON(http.StatusConflict, gin.H{"msg": "The subscription is already being stored in the database", "error type": customErr.Type(), "error": customErr.Error(), "body": res})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusCreated, gin.H{"msg": "The subscription successfully created", "body": res})
}

// @Summary Get subscription information
// @Description Gets subscription by its id or combination of user_id and service_name
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id query int false "1"
// @Param user_id query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Success 200 {object} models.Subscription
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /read [get]
func (h *Handler) Read(c *gin.Context) {
	// Getting query params
	id := c.DefaultQuery("id", "")
	userId := c.DefaultQuery("user_id", "")
	serviceName := c.DefaultQuery("service_name", "")
	if len(id) == 0 && (len(userId) == 0 || len(serviceName) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request"})
		return
	}

	// Getting subscription info from the database
	res, customErr := h.Db.Read(id, userId, serviceName)
	if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscription info from the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscrtiption was successfully read", "body": res})
}

// @Summary Update subscription
// @Description Updates existing subscription by its id
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
	var subscription Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Error while reading request's body", "error": err.Error()})
		return
	}

	// Validating service name
	if len(subscription.ServiceName) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Empty service name"})
		return
	}

	// Validating user id
	if _, err := uuid.Parse(subscription.UserId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id", "error": err.Error()})
		return
	}

	// Validating price
	if subscription.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid price"})
		return
	}

	customErr := h.Db.Update(subscription)
	if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while updating subscription info from the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscription was successfully updated"})
}

// @Summary Delete subscription
// @Description Deletes subscription by its id or combination of user_id and service_name
// @Tags subscriptions
// @Produce json
// @Param id query int false "1"
// @Param user_id query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Success 200
// @Failure 400
// @Failure 500
// @Router /delete [delete]
func (h *Handler) Delete(c *gin.Context) {
	// Getting query params
	id := c.DefaultQuery("id", "")
	userId := c.DefaultQuery("user_id", "")
	serviceName := c.DefaultQuery("service_name", "")
	if len(id) == 0 && (len(userId) == 0 || len(serviceName) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request"})
		return
	}

	customErr := h.Db.Delete(id, userId, serviceName)
	if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while deleting subscription info from the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscription is not found in the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subscription was successfully deleted"})
}

// @Summary Get list of subscriptions
// @Description Gest list of subscriptions filtered by user_id, service_name, start_date and end_date
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
// @Param service_name query string false "Yandex Plus"
// @Param start_date query string false "07-2025"
// @Param end_date query string false "08-2025"
// @Success 200 {array} models.Subscription
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /list [get]
func (h *Handler) List(c *gin.Context) {
	// Getting query params
	userId := c.DefaultQuery("user_id", "")
	serviceName := c.DefaultQuery("service_name", "")
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	res, customErr := h.Db.List(userId, serviceName, startDate, endDate)
	if customErr != nil && customErr.TypeOf == ErrTimeParse {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscriptions info from the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"msg": "The subscriptions are not found in the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The subcriptions were successfully read", "body": res})
}

// @Summary Get total sum of subscriptions prices
// @Description Get total amount of unique subscriptions and calculates its total price within the provided perio. It is implied that both start_date and end_date of subscription is being paid. The subscriptions can be filtered by user_id or service_name
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id query string false "60601fee-2bf1-4721-ae6f-7636e79a0cba"
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
	start := c.DefaultQuery("start_date", "")
	end := c.DefaultQuery("end_date", "")
	userId := c.DefaultQuery("user_id", "")
	serviceName := c.DefaultQuery("service_name", "")
	if len(start) <= 0 || len(end) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Empty time bounds"})
		return
	}

	res, customErr := h.Db.Summary(start, end, userId, serviceName)
	if customErr != nil && customErr.TypeOf == ErrInternalServer {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while getting subscription info from the database", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil && customErr.TypeOf == ErrTimeParse {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid time bounds", "error type": customErr.Type(), "error": customErr.Error()})
		return
	} else if customErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Unknown error", "error type": customErr.Type(), "error": customErr.Error()})
		return
	}

	// Writing response
	c.JSON(http.StatusOK, gin.H{"msg": "The total sum was successfully calculated", "body": res})
}
