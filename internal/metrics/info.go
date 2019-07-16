package metrics // import "github.com/pomerium/pomerium/internal/metrics"

import (
	"context"
	"runtime"
	"time"

	"github.com/pomerium/pomerium/internal/version"

	"github.com/pomerium/pomerium/internal/log"
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	//buildInfo               = stats.Int64("build_info", "Build Metadata", "1")
	configLastReload        = stats.Int64("config_last_reload_success_timestamp", "Timestamp of last successful config reload", "seconds")
	configLastReloadSuccess = stats.Int64("config_last_reload_success", "Returns 1 if last reload was successful", "1")

	// Metrics that may/must be measured or set up before the config is fully parsed
	// and metric views are initialized
	buildInfo      *metric.Int64Gauge
	policyCount    *metric.Int64DerivedGauge
	configChecksum *metric.Int64Gauge
	metricRegistry = metric.NewRegistry()

	// ConfigLastReloadView contains the timestamp the configuration was last reloaded, labeled by service
	ConfigLastReloadView = &view.View{
		Name:        configLastReload.Name(),
		Description: configLastReload.Description(),
		Measure:     configLastReload,
		TagKeys:     []tag.Key{keyService},
		Aggregation: view.LastValue(),
	}

	// ConfigLastReloadSuccessView contains the result of the last configuration reload, labeled by service
	ConfigLastReloadSuccessView = &view.View{
		Name:        configLastReloadSuccess.Name(),
		Description: configLastReloadSuccess.Description(),
		Measure:     configLastReloadSuccess,
		TagKeys:     []tag.Key{keyService},
		Aggregation: view.LastValue(),
	}
)

// SetBuildInfo records the pomerium build info.  You must call RegisterInfoMetrics to
// have this exported
func SetBuildInfo(service string) {
	if buildInfo == nil {
		buildInfoL, err := metricRegistry.AddInt64Gauge("build_info",
			metric.WithDescription("Build Metadata"),
			metric.WithLabelKeys("service", "version", "revision", "goversion"),
		)
		if err != nil {
			log.Error().Err(err).Msg("internal/metrics: failed to register build info metric")
			return
		}
		buildInfo = buildInfoL
	}

	m, err := buildInfo.GetEntry(
		metricdata.NewLabelValue(service),
		metricdata.NewLabelValue(version.FullVersion()),
		metricdata.NewLabelValue(version.GitCommit),
		metricdata.NewLabelValue((runtime.Version())),
	)
	if err != nil {
		log.Error().Err(err).Msg("internal/metrics: failed to add build info metric")
	}
	m.Set(1)
}

// SetConfigInfo records the status, checksum and timestamp of a configuration reload.  You must register InfoViews or the related
// config views before calling
func SetConfigInfo(service string, success bool, checksum string) {

	if success {
		serviceTag := tag.Insert(keyService, service)
		if err := stats.RecordWithTags(
			context.Background(),
			[]tag.Mutator{serviceTag},
			configLastReload.M(time.Now().Unix()),
		); err != nil {
			log.Error().Err(err).Msg("internal/metrics: failed to record config checksum timestamp")
		}

		if err := stats.RecordWithTags(
			context.Background(),
			[]tag.Mutator{serviceTag},
			configLastReloadSuccess.M(1),
		); err != nil {
			log.Error().Err(err).Msg("internal/metrics: failed to record config reload")
		}
	} else {
		stats.Record(context.Background(), configLastReloadSuccess.M(0))
	}
}

// Register non-view based metrics registry globally for export
func RegisterInfoMetrics() {
	metricproducer.GlobalManager().AddProducer(metricRegistry)
}

// SetConfigChecksum creates the configuration checksum metric.  You must call RegisterInfoMetrics to
// have this exported
func SetConfigChecksum(service string, checksum int64) {
	if configChecksum == nil {
		configChecksumL, err := metricRegistry.AddInt64Gauge("config_checksum_int64",
			metric.WithDescription("Config checksum represented in int64 notation"),
			metric.WithLabelKeys("service"),
		)
		if err != nil {
			log.Error().Err(err).Msg("internal/metrics: failed to register config checksum metric")
			return
		}
		configChecksum = configChecksumL
	}

	m, err := configChecksum.GetEntry(metricdata.NewLabelValue(service))
	if err != nil {
		log.Error().Err(err).Msg("internal/metrics: failed to add config checksum metric")
	}
	m.Set(checksum)
}

// AddPolicyCountCallback sets the function to call when exporting the
// policy count metric.   You must call RegisterInfoMetrics to have this
// exported
func AddPolicyCountCallback(service string, f func() int64) {
	if policyCount == nil {
		policyCountL, err := metricRegistry.AddInt64DerivedGauge("policy_count_total",
			metric.WithDescription("Total number of policies loaded"),
			metric.WithLabelKeys("service"),
		)
		if err != nil {
			log.Error().Err(err).Msg("internal/metrics: failed to register policy count metric")
			return
		}
		policyCount = policyCountL
	}

	err := policyCount.UpsertEntry(f, metricdata.NewLabelValue(service))
	if err != nil {
		log.Error().Err(err).Msg("internal/metrics: failed to add policy count metric")
	}
}
