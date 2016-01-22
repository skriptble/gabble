package bosh

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/namespace"
)

var body = element.New("body").AddAttr("xmlns", namespace.BOSH)
var badRequest = body.AddAttr("type", "terminate").AddAttr("condition", "bad-request")
