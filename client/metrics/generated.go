package metrics

// This file is generated by metricgen/gen.go.
import (
	"time"

	pb "github.com/sgielen/rufs/proto"
)

func SetClientStartTimeSeconds(circles []string, v time.Time) {
	setGauge(circles, pb.PushMetricsRequest_CLIENT_START_TIME_SECONDS, []string{}, float64(v.UnixNano()) / 1e9)
}

func SetTransferReadsActive(circles []string, v int64) {
	setGauge(circles, pb.PushMetricsRequest_TRANSFER_READS_ACTIVE, []string{}, float64(v))
}

func AddTransferOpens(circles []string, code string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_TRANSFER_OPENS, []string{code}, float64(v))
}

func AddTransferReads(circles []string, code string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_TRANSFER_READS, []string{code}, float64(v))
}

func AppendTransferReadSizes(circles []string, v float64) {
	appendDistribution(circles, pb.PushMetricsRequest_TRANSFER_READ_SIZES, []string{}, v)
}

func AppendTransferReadLatency(circles []string, code, recv_kbytes string, v float64) {
	appendDistribution(circles, pb.PushMetricsRequest_TRANSFER_READ_LATENCY, []string{code, recv_kbytes}, v)
}

func AddVfsFixedContentOpens(circles []string, basename string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_VFS_FIXED_CONTENT_OPENS, []string{basename}, float64(v))
}

func AddVfsReaddirs(circles []string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_VFS_READDIRS, []string{}, float64(v))
}

func AppendVfsReaddirLatency(circles []string, v float64) {
	appendDistribution(circles, pb.PushMetricsRequest_VFS_READDIR_LATENCY, []string{}, v)
}

func AddVfsPeerReaddirs(circles []string, peer, code string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_VFS_PEER_READDIRS, []string{peer, code}, float64(v))
}

func AppendVfsPeerReaddirLatency(circles []string, peer, code string, v float64) {
	appendDistribution(circles, pb.PushMetricsRequest_VFS_PEER_READDIR_LATENCY, []string{peer, code}, v)
}

func AddContentHashes(circles []string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_CONTENT_HASHES, []string{}, float64(v))
}

func AddContentRpcsRecv(circles []string, rpc, peer, code string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_CONTENT_RPCS_RECV, []string{rpc, peer, code}, float64(v))
}

func AppendContentRpcsRecvLatency(circles []string, rpc, peer, code string, v float64) {
	appendDistribution(circles, pb.PushMetricsRequest_CONTENT_RPCS_RECV_LATENCY, []string{rpc, peer, code}, v)
}

func AddContentOrchestrationJoined(circles []string, why string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_CONTENT_ORCHESTRATION_JOINED, []string{why}, float64(v))
}

func AddContentOrchestrationJoinFailed(circles []string, why string, v int64) {
	increaseCounter(circles, pb.PushMetricsRequest_CONTENT_ORCHESTRATION_JOIN_FAILED, []string{why}, float64(v))
}

func isCounter(t pb.PushMetricsRequest_MetricId) bool {
	switch t {
	case pb.PushMetricsRequest_TRANSFER_OPENS, pb.PushMetricsRequest_TRANSFER_READS, pb.PushMetricsRequest_VFS_FIXED_CONTENT_OPENS, pb.PushMetricsRequest_VFS_READDIRS, pb.PushMetricsRequest_VFS_PEER_READDIRS, pb.PushMetricsRequest_CONTENT_HASHES, pb.PushMetricsRequest_CONTENT_RPCS_RECV, pb.PushMetricsRequest_CONTENT_ORCHESTRATION_JOINED, pb.PushMetricsRequest_CONTENT_ORCHESTRATION_JOIN_FAILED:
		return true
	default:
		return false
	}
}

func isDistributionMetric(t pb.PushMetricsRequest_MetricId) bool {
	switch t {
	case pb.PushMetricsRequest_TRANSFER_READ_SIZES, pb.PushMetricsRequest_TRANSFER_READ_LATENCY, pb.PushMetricsRequest_VFS_READDIR_LATENCY, pb.PushMetricsRequest_VFS_PEER_READDIR_LATENCY, pb.PushMetricsRequest_CONTENT_RPCS_RECV_LATENCY:
		return true
	default:
		return false
	}
}
