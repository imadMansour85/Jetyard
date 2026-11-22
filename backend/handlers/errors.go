package handlers

import (
	"errors"
	"net/http"

	natsgo "github.com/nats-io/nats.go"
)

// notFoundErrors are the NATS SDK sentinels that mean "the thing you asked
// for doesn't exist". The client wraps them with %w, so errors.Is sees them.
var notFoundErrors = []error{
	natsgo.ErrStreamNotFound,
	natsgo.ErrConsumerNotFound,
	natsgo.ErrBucketNotFound,
	natsgo.ErrKeyNotFound,
	natsgo.ErrMsgNotFound,
}

// badRequestErrors are client-side validation errors the SDK returns before
// it sends anything to the server, i.e. bad user input.
var badRequestErrors = []error{
	natsgo.ErrStreamNameRequired,
	natsgo.ErrInvalidStreamName,
	natsgo.ErrConsumerNameRequired,
	natsgo.ErrInvalidConsumerName,
	natsgo.ErrInvalidBucketName,
	natsgo.ErrInvalidKey,
	natsgo.ErrBadSubject,
	natsgo.ErrDuplicateFilterSubjects,
}

// natsErrorStatus maps an error from the NATS client to an HTTP status:
// missing resources are 404, invalid input is 400, JetStream API rejections
// keep their 4xx code (e.g. 400 for overlapping subjects), everything else is
// a 500.
func natsErrorStatus(err error) int {
	for _, target := range notFoundErrors {
		if errors.Is(err, target) {
			return http.StatusNotFound
		}
	}
	for _, target := range badRequestErrors {
		if errors.Is(err, target) {
			return http.StatusBadRequest
		}
	}
	var apiErr *natsgo.APIError
	if errors.As(err, &apiErr) && apiErr.Code >= 400 && apiErr.Code < 500 {
		return apiErr.Code
	}
	return http.StatusInternalServerError
}

// writeNATSError writes err with the status natsErrorStatus picks for it.
func writeNATSError(w http.ResponseWriter, err error) {
	writeError(w, natsErrorStatus(err), err.Error())
}
