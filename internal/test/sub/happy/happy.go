package happy

import (
	"github.com/kevinconway/wrapgen/v2/internal/test/happy"
)

type DemoType struct{}

type Demo interface {
	Make(param happy.ExportedStruct, second DemoType) happy.NonInterfaceAlias
}
