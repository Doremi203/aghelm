package runtime

import (
	"github.com/Doremi203/aghelm/internal/config"
	"github.com/Doremi203/aghelm/internal/systheme"
	"github.com/Doremi203/aghelm/internal/ui/effects"
	"maps"
)

type Services struct{}
type ProbeRequest struct {
	Target   uint64
	Commands map[string]string
}
type ProbeResult struct {
	Installed map[string]bool
	Scheme    systheme.Scheme
}

func (ProbeRequest) EffectName() string            { return "onboarding-probe" }
func (r ProbeRequest) EffectCopy() effects.Request { r.Commands = maps.Clone(r.Commands); return r }
func (ProbeResult) EffectResult()                  {}
func (r ProbeRequest) Run(Services) (effects.Result, error) {
	installed := make(map[string]bool, len(r.Commands))
	for name, command := range r.Commands {
		installed[name] = config.CheckInstalled(command) == nil
	}
	return ProbeResult{Installed: installed, Scheme: systheme.Detect()}, nil
}
