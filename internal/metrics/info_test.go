package metrics // import "github.com/pomerium/pomerium/internal/metrics"

import (
	"runtime"
	"testing"

	"github.com/pomerium/pomerium/internal/version"

	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
)

func Test_SetConfigInfo(t *testing.T) {
	tests := []struct {
		name                  string
		success               bool
		checksum              string
		wantLastReload        string
		wantLastReloadSuccess string
	}{
		{"success", true, "abcde", "{ { {service test_service} }&{1.", "{ { {service test_service} }&{1} }"},
		{"failed", false, "abcde", "", "{ {  }&{0} }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UnRegisterView(InfoViews)
			RegisterView(InfoViews)

			SetConfigInfo("test_service", tt.success, tt.checksum)

			testDataRetrieval(ConfigLastReloadView, t, tt.wantLastReload)
			testDataRetrieval(ConfigLastReloadSuccessView, t, tt.wantLastReloadSuccess)
		})
	}
}

func Test_SetBuildInfo(t *testing.T) {
	metricRegistry = metric.NewRegistry()
	defer func() { metricRegistry = metric.NewRegistry() }()

	version.Version = "v0.0.1"
	version.GitCommit = "deadbeef"

	wantLabels := []metricdata.LabelValue{
		{Value: "test_service", Present: true},
		{Value: version.FullVersion(), Present: true},
		{Value: version.GitCommit, Present: true},
		{Value: runtime.Version(), Present: true},
	}

	SetBuildInfo("test_service")
	testMetricRetrieval(metricRegistry.Read(), t, wantLabels, 1)
}

func Test_AddPolicyCountCallback(t *testing.T) {
	metricRegistry = metric.NewRegistry()
	defer func() { metricRegistry = metric.NewRegistry() }()

	wantValue := int64(42)
	wantLabels := []metricdata.LabelValue{{Value: "test_service", Present: true}}
	AddPolicyCountCallback("test_service", func() int64 { return wantValue })

	testMetricRetrieval(metricRegistry.Read(), t, wantLabels, wantValue)
}

func Test_SetConfigChecksum(t *testing.T) {
	metricRegistry = metric.NewRegistry()
	defer func() { metricRegistry = metric.NewRegistry() }()

	wantValue := int64(42)
	wantLabels := []metricdata.LabelValue{{Value: "test_service", Present: true}}
	SetConfigChecksum("test_service", wantValue)

	testMetricRetrieval(metricRegistry.Read(), t, wantLabels, wantValue)
}

func Test_RegisterInfoMetrics(t *testing.T) {
	metricproducer.GlobalManager().DeleteProducer(metricRegistry)
	RegisterInfoMetrics()
	r := metricproducer.GlobalManager().GetAll()
	if len(r) != 2 {
		t.Error("Did not find enough registries")
	}
}
