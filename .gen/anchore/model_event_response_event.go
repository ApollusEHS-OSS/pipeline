/*
 * Anchore Engine API Server
 *
 * This is the Anchore Engine API. Provides the primary external API for users of the service.
 *
 * API version: 0.1.12
 * Contact: nurmi@anchore.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package anchore
import (
	"time"
)

type EventResponseEvent struct {
	Resource EventResponseEventResource `json:"resource,omitempty"`
	Level string `json:"level,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Source EventResponseEventSource `json:"source,omitempty"`
	Type string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}