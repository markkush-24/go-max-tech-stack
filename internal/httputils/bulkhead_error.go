package httputils

import "errors"

var ErrBulkheadRejected = errors.New("bulkhead rejected")
