package ffvm

import "regexp"

var tagRegex = regexp.MustCompile(`^\s*(v:.*,m:.*|m:.*,v:.*|m:.*|v:.*)+\s*$`)
