package stats_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/upfluence/stats"
)

func ExampleRootScope() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)
	counter := scope.Counter("requests_total")
	counter.Inc()

	snapshot := collector.Get()
	fmt.Printf("%s: %d\n", snapshot.Counters[0].Name, snapshot.Counters[0].Value)
	// Output: requests_total: 1
}

func ExampleScope_Counter() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	counter := scope.Counter("http_requests_total")
	counter.Inc()
	counter.Add(5)

	snapshot := collector.Get()
	fmt.Printf("%s: %d\n", snapshot.Counters[0].Name, snapshot.Counters[0].Value)
	// Output: http_requests_total: 6
}

func ExampleScope_CounterVector() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	counterVec := scope.CounterVector("http_requests_total", []string{"method", "status"})
	counterVec.WithLabels("GET", "200").Inc()
	counterVec.WithLabels("POST", "201").Add(10)

	snapshot := collector.Get()
	for _, c := range snapshot.Counters {
		fmt.Printf("%s{method=%s, status=%s}: %d\n",
			c.Name, c.Labels["method"], c.Labels["status"], c.Value)
	}
	// Output:
	// http_requests_total{method=GET, status=200}: 1
	// http_requests_total{method=POST, status=201}: 10
}

func ExampleScope_Gauge() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	gauge := scope.Gauge("memory_usage_bytes")
	gauge.Update(1024000)

	snapshot := collector.Get()
	fmt.Printf("%s: %d\n", snapshot.Gauges[0].Name, snapshot.Gauges[0].Value)
	// Output: memory_usage_bytes: 1024000
}

func ExampleScope_GaugeVector() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	gaugeVec := scope.GaugeVector("cpu_usage_percent", []string{"core"})
	gaugeVec.WithLabels("0").Update(75)
	gaugeVec.WithLabels("1").Update(82)

	snapshot := collector.Get()
	for _, g := range snapshot.Gauges {
		fmt.Printf("%s{core=%s}: %d\n", g.Name, g.Labels["core"], g.Value)
	}
	// Output:
	// cpu_usage_percent{core=0}: 75
	// cpu_usage_percent{core=1}: 82
}

func ExampleScope_Histogram() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	histogram := scope.Histogram("response_time_seconds")
	histogram.Record(0.234)
	histogram.Record(0.123)

	snapshot := collector.Get()
	fmt.Printf("%s: count=%d, sum=%.3f\n",
		snapshot.Histograms[0].Name,
		snapshot.Histograms[0].Value.Count,
		snapshot.Histograms[0].Value.Sum)
	// Output: response_time_seconds: count=2, sum=0.357
}

func ExampleScope_HistogramVector() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	histVec := scope.HistogramVector("request_duration_seconds", []string{"endpoint", "method"})
	histVec.WithLabels("/api/users", "GET").Record(0.045)
	histVec.WithLabels("/api/users", "GET").Record(0.032)

	snapshot := collector.Get()
	h := snapshot.Histograms[0]
	fmt.Printf("%s{endpoint=%s, method=%s}: count=%d\n",
		h.Name, h.Value.Tags["endpoint"], h.Value.Tags["method"], h.Value.Count)
	// Output: request_duration_seconds{endpoint=/api/users, method=GET}: count=2
}

func ExampleStaticBuckets() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	histogram := scope.Histogram("request_duration_seconds",
		stats.StaticBuckets([]float64{0.01, 0.05, 0.1, 0.5, 1.0}))
	histogram.Record(0.123)

	snapshot := collector.Get()
	fmt.Printf("%s: %d buckets\n",
		snapshot.Histograms[0].Name,
		len(snapshot.Histograms[0].Value.Buckets))
	// Output: request_duration_seconds: 6 buckets
}

func ExampleScope_Scope() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	dbScope := scope.Scope("database", map[string]string{
		"service": "user-api",
		"env":     "production",
	})

	counter := dbScope.Counter("queries")
	counter.Inc()

	snapshot := collector.Get()
	c := snapshot.Counters[0]
	fmt.Printf("%s{service=%s, env=%s}: %d\n",
		c.Name, c.Labels["service"], c.Labels["env"], c.Value)
	// Output: database_queries{service=user-api, env=production}: 1
}

func ExampleNewTimer() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	timer := stats.NewTimer(scope, "operation_duration")
	stopwatch := timer.Start()
	// ... perform operation ...
	stopwatch.Stop()

	snapshot := collector.Get()
	fmt.Printf("%s: count=%d\n",
		snapshot.Histograms[0].Name,
		snapshot.Histograms[0].Value.Count)
	// Output: operation_duration_seconds: count=1
}

func ExampleWithTimerSuffix() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	timer := stats.NewTimer(scope, "request_duration",
		stats.WithTimerSuffix("_ms"))
	stopwatch := timer.Start()
	stopwatch.Stop()

	snapshot := collector.Get()
	fmt.Printf("%s\n", snapshot.Histograms[0].Name)
	// Output: request_duration_ms
}

func ExampleNewInstrument() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	instrument := stats.NewInstrument(scope, "api_request")

	err := instrument.Exec(func() error {
		// Your operation here
		return nil
	})

	if err != nil {
		fmt.Println("error:", err)
	}

	snapshot := collector.Get()
	fmt.Printf("Counters: %d, Histograms: %d\n",
		len(snapshot.Counters), len(snapshot.Histograms))
	// Output: Counters: 2, Histograms: 1
}

func ExampleNewInstrumentVector() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	instVec := stats.NewInstrumentVector(scope, "database_query", []string{"table", "operation"})

	err := instVec.WithLabels("users", "select").Exec(func() error {
		// Perform database query
		return nil
	})

	if err != nil {
		fmt.Println("error:", err)
	}

	snapshot := collector.Get()
	for _, c := range snapshot.Counters {
		if c.Labels["status"] != "" {
			fmt.Printf("%s{table=%s, operation=%s, status=%s}: %d\n",
				c.Name, c.Labels["table"], c.Labels["operation"], c.Labels["status"], c.Value)
		}
	}
	// Output: database_query_total{table=users, operation=select, status=success}: 1
}

func ExampleDisableStartedCounter() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	inst := stats.NewInstrument(scope, "task",
		stats.DisableStartedCounter())

	_ = inst.Exec(func() error {
		return nil
	})

	snapshot := collector.Get()
	fmt.Printf("Counters: %d\n", len(snapshot.Counters))
	// Output: Counters: 1
}

func ExampleWithFormatter() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	formatter := func(err error) string {
		if err == nil {
			return "success"
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return "timeout"
		}
		return "error"
	}

	inst := stats.NewInstrument(scope, "api",
		stats.WithFormatter(formatter))

	_ = inst.Exec(func() error {
		return context.DeadlineExceeded
	})

	snapshot := collector.Get()
	for _, c := range snapshot.Counters {
		if c.Name == "api_total" {
			fmt.Printf("status=%s: %d\n", c.Labels["status"], c.Value)
		}
	}
	// Output: status=timeout: 1
}

func ExampleWithCounterLabel() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	inst := stats.NewInstrument(scope, "request",
		stats.WithCounterLabel("result"))

	_ = inst.Exec(func() error {
		return nil
	})

	snapshot := collector.Get()
	for _, c := range snapshot.Counters {
		if c.Name == "request_total" {
			fmt.Printf("%s{result=%s}: %d\n", c.Name, c.Labels["result"], c.Value)
		}
	}
	// Output: request_total{result=success}: 1
}

func ExampleWithTimerOptions() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	inst := stats.NewInstrument(scope, "query",
		stats.WithTimerOptions(
			stats.WithTimerSuffix("_ms"),
			stats.WithHistogramOptions(
				stats.StaticBuckets([]float64{1, 10, 100}))))

	_ = inst.Exec(func() error {
		return nil
	})

	snapshot := collector.Get()
	fmt.Printf("%s: %d buckets\n",
		snapshot.Histograms[0].Name,
		len(snapshot.Histograms[0].Value.Buckets))
	// Output: query_duration_ms: 4 buckets
}

func ExampleGlobalIncarnationScope() {
	collector := stats.NewStaticCollector()
	scope := stats.GlobalIncarnationScope(stats.RootScope(collector))

	counter := scope.Counter("requests")
	counter.Inc()

	snapshot := collector.Get()
	fmt.Printf("%s{incarnation=%s}: %d\n",
		snapshot.Counters[0].Name,
		snapshot.Counters[0].Labels["incarnation"],
		snapshot.Counters[0].Value)
	// Output: requests{incarnation=0}: 1
}

func ExampleLocalIncarnationScope() {
	collector := stats.NewStaticCollector()
	scope := stats.LocalIncarnationScope(stats.RootScope(collector), "version")

	counter := scope.Counter("requests")
	counter.Inc()

	snapshot := collector.Get()
	fmt.Printf("%s{version=%s}: %d\n",
		snapshot.Counters[0].Name,
		snapshot.Counters[0].Labels["version"],
		snapshot.Counters[0].Value)
	// Output: requests{version=0}: 1
}

func ExampleNewStaticCollector() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	counter := scope.Counter("requests")
	counter.Inc()

	snapshot := collector.Get()
	fmt.Printf("Counters: %d\n", len(snapshot.Counters))
	// Output: Counters: 1
}

func ExampleExecInstrument2() {
	collector := stats.NewStaticCollector()
	scope := stats.RootScope(collector)

	inst := stats.NewInstrument(scope, "fetch_user")

	result, _ := stats.ExecInstrument2(inst, func() (string, error) {
		// Simulate fetching user data
		return "user123", nil
	})

	fmt.Printf("Result: %s\n", result)
	// Output: Result: user123
}
