package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/sgielen/rufs/proto"
	strcase "github.com/stoewer/go-strcase"
)

func main() {
	serverCode := flag.Bool("server", false, "Whether to generate code for the discovery server (or client)")
	flag.Parse()
	var out bytes.Buffer
	out.WriteString("package metrics\n")
	out.WriteString("\n")
	out.WriteString("// This file is generated by metricgen/gen.go.")
	out.WriteString("\n")

	if *serverCode {
		serverMetrics(&out)
	} else {
		clientMetrics(&out)
	}

	fn := flag.Arg(0)
	if err := ioutil.WriteFile(fn, out.Bytes(), 0644); err != nil {
		log.Fatalf("Failed to write to %q: %v", fn, err)
	}
}

type metric struct {
	name       string
	help       string
	metricType pb.PushMetricsRequest_MetricType
	fields     []string
}

func listMetrics() []metric {
	var ret []metric
	vs := pb.PushMetricsRequest_UNKNOWN.Descriptor().Values()
	for i := 0; vs.Len() > i; i++ {
		ev := vs.Get(i)
		if ev.Number() == pb.PushMetricsRequest_UNKNOWN.Number() {
			continue
		}
		ed := ev.Options().(*dpb.EnumValueOptions)
		mt, err := proto.GetExtension(ed, pb.E_PushMetricsRequest_MetricType)
		if err != nil {
			log.Fatalf("Failed to get metric type for %s: %v", ev.Name(), err)
		}
		fs, err := proto.GetExtension(ed, pb.E_PushMetricsRequest_MetricFields)
		if err != nil {
			fs = []string{}
		}
		help, err := proto.GetExtension(ed, pb.E_PushMetricsRequest_MetricDescription)
		if err != nil {
			var empty string
			help = &empty
		}
		ret = append(ret, metric{
			name:       string(ev.Name()),
			help:       *help.(*string),
			metricType: *(mt.(*pb.PushMetricsRequest_MetricType)),
			fields:     fs.([]string),
		})
	}
	return ret
}

func serverMetrics(out *bytes.Buffer) {
	out.WriteString("import (\n")
	out.WriteString("	\"github.com/prometheus/client_golang/prometheus\"\n")
	out.WriteString("	pb \"github.com/sgielen/rufs/proto\"\n")
	out.WriteString(")\n")
	out.WriteString("\n")
	out.WriteString("var (\n")
	out.WriteString("	metrics = map[pb.PushMetricsRequest_MetricId]processMetric{\n")
loop:
	for _, m := range listMetrics() {
		labels := "nil"
		if len(m.fields) > 0 {
			labels = `[]string{"` + strings.Join(m.fields, `", "`) + `"}`
		}
		constructor := ""
		switch m.metricType {
		case pb.PushMetricsRequest_TIME_GAUGE, pb.PushMetricsRequest_INT64_GAUGE:
			constructor = "newGauge(prometheus.GaugeOpts"
		case pb.PushMetricsRequest_COUNTER:
			constructor = "newCounter(prometheus.CounterOpts"
		case pb.PushMetricsRequest_DISTRIBUTION:
			constructor = "newHistogram(prometheus.HistogramOpts"
		default:
			out.WriteString("\n")
			fmt.Fprintf(out, "// Unsupported metric type %s\n", m.metricType)
			continue loop
		}
		fmt.Fprintf(out, "		pb.PushMetricsRequest_%s: %s{\n", m.name, constructor)
		out.WriteString("			Namespace: \"rufs\",\n")
		fmt.Fprintf(out, "			Name: %q,\n", strcase.SnakeCase(m.name))
		fmt.Fprintf(out, "			Help: %q,\n", m.help)
		if m.metricType == pb.PushMetricsRequest_DISTRIBUTION {
			fmt.Fprintf(out, "			Buckets: bucketsFor%s,\n", strcase.UpperCamelCase(m.name))
		}
		fmt.Fprintf(out, "		}, %s),\n", labels)
	}
	out.WriteString("	}\n")
	out.WriteString(")\n")
}

func clientMetrics(out *bytes.Buffer) {
	out.WriteString("import (\n")
	out.WriteString("	\"time\"\n")
	out.WriteString("\n")
	out.WriteString("	pb \"github.com/sgielen/rufs/proto\"\n")
	out.WriteString(")\n")

	var counterMetrics, distributionMetrics []string

loop:
	for _, m := range listMetrics() {
		fieldsSignature := ""
		if len(m.fields) > 0 {
			fieldsSignature = " " + strings.Join(m.fields, ", ") + " string,"
		}
		arg := "v int64"
		transformation := "float64(v)"
		namePrefix := "E R R O R"
		internalFunction := " E R R O R"
		switch m.metricType {
		case pb.PushMetricsRequest_TIME_GAUGE:
			namePrefix = "Set"
			internalFunction = "setGauge"
			arg = "v time.Time"
			transformation = "float64(v.UnixNano()) / 1000.0"
		case pb.PushMetricsRequest_INT64_GAUGE:
			namePrefix = "Set"
			internalFunction = "setGauge"
		case pb.PushMetricsRequest_COUNTER:
			namePrefix = "Add"
			internalFunction = "increaseCounter"
			counterMetrics = append(counterMetrics, fmt.Sprintf("pb.PushMetricsRequest_%s", m.name))
		case pb.PushMetricsRequest_DISTRIBUTION:
			namePrefix = "Append"
			internalFunction = "appendDistribution"
			arg = "v float64"
			transformation = "v"
			distributionMetrics = append(distributionMetrics, fmt.Sprintf("pb.PushMetricsRequest_%s", m.name))
		default:
			out.WriteString("\n")
			fmt.Fprintf(out, "// Unsupported metric type %s\n", m.metricType)
			continue loop
		}

		out.WriteString("\n")
		fmt.Fprintf(out, "func %s%s(circles []string,%s %s) {\n", namePrefix, strcase.UpperCamelCase(m.name), fieldsSignature, arg)
		fmt.Fprintf(out, "	%s(circles, pb.PushMetricsRequest_%s, []string{%s}, %s)\n", internalFunction, m.name, strings.Join(m.fields, ", "), transformation)
		out.WriteString("}\n")
	}

	out.WriteString("\n")
	out.WriteString("func isCounter(t pb.PushMetricsRequest_MetricId) bool {\n")
	out.WriteString("	switch t {\n")
	fmt.Fprintf(out, "	case %s:\n", strings.Join(counterMetrics, ", "))
	out.WriteString("		return true\n")
	out.WriteString("	default:\n")
	out.WriteString("		return false\n")
	out.WriteString("	}\n")
	out.WriteString("}\n")

	out.WriteString("\n")
	out.WriteString("func isDistributionMetric(t pb.PushMetricsRequest_MetricId) bool {\n")
	out.WriteString("	switch t {\n")
	fmt.Fprintf(out, "	case %s:\n", strings.Join(distributionMetrics, ", "))
	out.WriteString("		return true\n")
	out.WriteString("	default:\n")
	out.WriteString("		return false\n")
	out.WriteString("	}\n")
	out.WriteString("}\n")
}
