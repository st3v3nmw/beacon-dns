package dns

import (
	"github.com/maypok86/otter"
	dnslib "github.com/miekg/dns"
)

var (
	Cache otter.CacheWithVariableTTL[string, *dnslib.Msg]
)

func NewCache() error {
	var err error
	Cache, err = otter.MustBuilder[string, *dnslib.Msg](1_048_576).
		CollectStats().
		WithVariableTTL().
		Build()
	if err != nil {
		return err
	}

	return nil
}
