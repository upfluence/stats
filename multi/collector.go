package multi

import "github.com/upfluence/stats"

type multiCollector []stats.Collector

func (cs multiCollector) Close() error {
	for _, c := range cs {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (cs multiCollector) RegisterCounter(n string, g stats.Int64VectorGetter) {
	for _, c := range cs {
		c.RegisterCounter(n, g)
	}
}

func (cs multiCollector) RegisterGauge(n string, g stats.Int64VectorGetter) {
	for _, c := range cs {
		c.RegisterGauge(n, g)
	}
}

func WrapCollectors(cs ...stats.Collector) stats.Collector {
	switch len(cs) {
	case 0:
		return nil
	case 1:
		return cs[0]
	}

	return multiCollector(cs)
}
