package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/ojcm/prometer/internal/metrics"
	"github.com/olivercullimore/geo-energy-data-client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	powerTypeElectricity = "ELECTRICITY"
	powerTypeGas         = "GAS_ENERGY"

	durationDay   = "DAY"
	durationWeek  = "WEEK"
	durationMonth = "MONTH"
)

type config struct {
	User         string        `env:"GEO_USER"`
	Pass         string        `env:"GEO_PASS"`
	PollInterval time.Duration `env:"POLL_INTERNVAL" envDefault:"10s"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		logger.Error(fmt.Sprintf("failed to parse config: %s", err))
		return
	}

	accessToken, err := geo.GetAccessToken(cfg.User, cfg.Pass)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get access token config: %s", err))
		return
	}

	deviceData, err := geo.GetDeviceData(accessToken)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get devices: %s", err))
		return
	}

	if numDevices := len(deviceData.SystemDetails); numDevices != 1 {
		logger.Error(fmt.Sprintf("expected 1 device got %d", numDevices))
		return
	}

	geoSystemID := deviceData.SystemDetails[0].SystemID

	go pollReadings(logger, accessToken, geoSystemID)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9090", nil)
}

func pollReadings(logger *slog.Logger, accessToken, geoSystemID string) {
	const pollInterval = 10 * time.Second // match Prometheus scrape interval

	logger.Info("starting poller")

	tick := time.Tick(pollInterval)

	// Set metrics immediately then on each tick
	for {
		err := setLiveReadings(accessToken, geoSystemID)
		if err != nil {
			logger.Error(err.Error())
		}

		if err := setPeriodicData(accessToken, geoSystemID); err != nil {
			logger.Error(err.Error())
		}

		<-tick
	}
}

func setLiveReadings(accessToken, geoSystemID string) error {
	// Get live meter data
	liveData, err := geo.GetLiveMeterData(accessToken, geoSystemID)
	if err != nil {
		return fmt.Errorf("getting live data: %w", err)
	}

	for _, power := range liveData.Power {
		switch power.Type {
		case powerTypeElectricity:
			metrics.LiveUsage(metrics.Electricity, power.Watts)
		case powerTypeGas:
			metrics.LiveUsage(metrics.Gas, power.Watts)
		}
	}
	return nil
}

func setPeriodicData(accessToken, geoSystemID string) error {
	// Get periodic meter data
	periodicData, err := geo.GetPeriodicMeterData(accessToken, geoSystemID)
	if err != nil {
		return fmt.Errorf("getting data: %w", err)
	}

	for _, power := range periodicData.TotalConsumptionList {
		if !power.ValueAvailable {
			continue
		}

		switch power.CommodityType {
		case powerTypeElectricity:
			metrics.MeterReading(metrics.Electricity, power.TotalConsumption, time.Unix(power.ReadingTime, 0))
		case powerTypeGas:
			metrics.MeterReading(metrics.Gas, power.TotalConsumption, time.Unix(power.ReadingTime, 0))
		}
	}

	for _, c := range periodicData.CurrentCostsElec {
		switch c.Duration {
		case durationDay:
			metrics.Cost(metrics.Electricity, metrics.Day, c.CostAmount)
		case durationWeek:
			metrics.Cost(metrics.Electricity, metrics.Week, c.CostAmount)
		case durationMonth:
			metrics.Cost(metrics.Electricity, metrics.Month, c.CostAmount)
		}
	}

	metrics.CostDelay(metrics.Electricity, time.Unix(periodicData.CurrentCostsElecTimestamp, 0))

	for _, c := range periodicData.CurrentCostsGas {
		switch c.Duration {
		case durationDay:
			metrics.Cost(metrics.Gas, metrics.Day, c.CostAmount)
		case durationWeek:
			metrics.Cost(metrics.Gas, metrics.Week, c.CostAmount)
		case durationMonth:
			metrics.Cost(metrics.Gas, metrics.Month, c.CostAmount)
		}
	}

	metrics.CostDelay(metrics.Electricity, time.Unix(periodicData.CurrentCostsGasTimestamp, 0))

	return nil
}
