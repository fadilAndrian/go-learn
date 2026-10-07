package merchant

import "errors"

// Semua error milik modul merchant. Handler mengubahnya jadi status HTTP di toHTTP.
var ErrNotFound = errors.New("merchant not found")
