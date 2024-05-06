package otel

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/trace"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/utils"
)

type Option func(l logMessage) logMessage

type logMessage struct {
	ID          string `json:"ID"`
	ServiceName string `json:"serviceName"`
	TraceInfo   struct {
		TraceID string `json:"traceID"`
		SpanID  string `json:"spanID"`
	}
	UserInfo struct {
		UserID   string `json:"userID"`
		UserUUID string `json:"userUUID"`
	}
	Event struct {
		IsAuditEvent bool `json:"isAuditEvent"`
		EventInfo    struct {
			EventName string `json:"eventName"`
			Billable  bool   `json:"billable"`
		}
		EventResource any    `json:"eventResource"`
		EventResult   any    `json:"eventResult"`
		EventMessage  string `json:"eventMessage"`
	}
	ErrorMessage string `json:"errorMessage"`
	Metadata     any
}

func SetEventResource(res any) Option {
	return func(l logMessage) logMessage {
		l.Event.EventResource = res
		return l
	}
}

func SetEventResult(result any) Option {
	return func(l logMessage) logMessage {
		l.Event.EventResult = result
		return l
	}
}

func SetEventMessage(message string) Option {
	return func(l logMessage) logMessage {
		l.Event.EventMessage = message
		return l
	}
}

func SetErrorMessage(e string) Option {
	return func(l logMessage) logMessage {
		l.ErrorMessage = e
		return l
	}
}

func SetMetadata(m string) Option {
	return func(l logMessage) logMessage {
		l.Metadata = m
		return l
	}
}

func NewLogMessage(
	ctx context.Context,
	span trace.Span,
	logID string,
	eventName string,
	options ...Option,
) []byte {
	logMessage := logMessage{}
	logMessage.ID = logID
	logMessage.ServiceName = "model-backend"
	logMessage.TraceInfo = struct {
		TraceID string "json:\"traceID\""
		SpanID  string "json:\"spanID\""
	}{
		TraceID: span.SpanContext().TraceID().String(),
		SpanID:  span.SpanContext().SpanID().String(),
	}
	logMessage.UserInfo = struct {
		UserID   string "json:\"userID\""
		UserUUID string "json:\"userUUID\""
	}{
		UserUUID: resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey),
	}
	logMessage.Event = struct {
		IsAuditEvent bool "json:\"isAuditEvent\""
		EventInfo    struct {
			EventName string "json:\"eventName\""
			Billable  bool   "json:\"billable\""
		}
		EventResource any    "json:\"eventResource\""
		EventResult   any    "json:\"eventResult\""
		EventMessage  string "json:\"eventMessage\""
	}{
		IsAuditEvent: utils.IsAuditEvent(eventName),
		EventInfo: struct {
			EventName string "json:\"eventName\""
			Billable  bool   "json:\"billable\""
		}{
			EventName: eventName,
			Billable:  utils.IsBillableEvent(eventName),
		},
	}

	for _, o := range options {
		logMessage = o(logMessage)
	}

	bLogMessage, _ := json.Marshal(logMessage)

	return bLogMessage
}
