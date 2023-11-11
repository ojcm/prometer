package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Utility string

const (
	Electricity Utility = "electricity"
	Gas         Utility = "gas"
)

type Duration string

const (
	Day   Duration = "day"
	Week  Duration = "week"
	Month Duration = "month"
)

var (
	liveUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "live_watts",
	}, []string{"utility"})

	meterReading = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "meter_reading",
	}, []string{"utility"})
	meterReadingDelay = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "meter_reading_delay_seconds",
	}, []string{"utility"})

	cost = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "current_cost_pence",
	}, []string{"utility", "duration"})
	costDelay = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "current_cost_delay_seconds",
	}, []string{"utility"})
)

func LiveUsage(utility Utility, valueWatts float64) {
	liveUsage.WithLabelValues(string(utility)).Set(valueWatts)
}

func MeterReading(utility Utility, valueWatts float64, readingTime time.Time) {
	meterReading.WithLabelValues(string(utility)).Set(valueWatts)
	meterReadingDelay.WithLabelValues(string(utility)).Set(time.Since(readingTime).Seconds())
}

func Cost(utility Utility, chargePeriod Duration, pricePence float64) {
	cost.WithLabelValues(string(utility), string(chargePeriod)).Set(pricePence)
}

func CostDelay(utility Utility, readingTime time.Time) {
	costDelay.WithLabelValues(string(utility)).Set(time.Since(readingTime).Seconds())
}
