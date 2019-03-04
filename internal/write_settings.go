package internal

import (
	"github.com/naveego/plugin-oracle/internal/pub"
)


// Settings object for write requests
// Contains target schema and the commit sla timeout
type WriteSettings struct {
	Schema		*pub.Schema   `json:"schema"`
	CommitSLA	int32		  `json:"commitSla"`
}
