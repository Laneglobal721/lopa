package passive

import (
	"github.com/yanjiulab/lopa/internal/passive/source"
)

// Delta computes the difference (current - prev) for each counter.
// If current < prev (e.g. counter reset), that field is treated as 0.
func Delta(prev, current *source.InterfaceSnapshot) *source.InterfaceSnapshot {
	d := &source.InterfaceSnapshot{}
	if current.BytesIn >= prev.BytesIn {
		d.BytesIn = current.BytesIn - prev.BytesIn
	}
	if current.BytesOut >= prev.BytesOut {
		d.BytesOut = current.BytesOut - prev.BytesOut
	}
	if current.PacketsIn >= prev.PacketsIn {
		d.PacketsIn = current.PacketsIn - prev.PacketsIn
	}
	if current.PacketsOut >= prev.PacketsOut {
		d.PacketsOut = current.PacketsOut - prev.PacketsOut
	}
	if current.ErrorsIn >= prev.ErrorsIn {
		d.ErrorsIn = current.ErrorsIn - prev.ErrorsIn
	}
	if current.ErrorsOut >= prev.ErrorsOut {
		d.ErrorsOut = current.ErrorsOut - prev.ErrorsOut
	}
	if current.DropsIn >= prev.DropsIn {
		d.DropsIn = current.DropsIn - prev.DropsIn
	}
	if current.DropsOut >= prev.DropsOut {
		d.DropsOut = current.DropsOut - prev.DropsOut
	}
	return d
}
