package solver

import (
	"regexp"
	"strconv"
)


type IncomingRequest struct {
	DataProviders []string    `json:"dataProviders"`
	DataRequest   DataRequest `json:"data_request"`
}

type DataRequest struct {
	Query   string         `json:"query"`
	Options RequestOptions `json:"options"`
}

type RequestOptions struct {
	Graph     bool `json:"graph"`
	Aggregate bool `json:"aggregate"`
}


func BuildRequestContext(input IncomingRequest) RequestContext {
	limit, hasLimit := ExtractSQLLimit(input.DataRequest.Query)

	extraHops := 0
	if input.DataRequest.Options.Graph {
		extraHops++
	}
	if input.DataRequest.Options.Aggregate {
		extraHops++
	}

	return RequestContext{
		SQLLimit:            limit,
		HasSQLLimit:         hasLimit,
		OptionalServiceHops: extraHops,
		DataProviderCount:   len(input.DataProviders),
	}
}


func ExtractSQLLimit(query string) (int, bool) {
	re := regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)
	match := re.FindStringSubmatch(query)

	if len(match) < 2 {
		return 0, false
	}

	limit, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return limit, true
}
