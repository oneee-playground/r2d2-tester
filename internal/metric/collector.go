package metric

import (
	"bufio"
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type Stat struct {
	Container string

	TotalCPUUsage   float64
	CPUUsagePerCore []float64
	MemoryUsage     float64

	BlockRead, BlockWrite uint64
	NetRead, NetWrite     uint64
}

type Collector struct {
	Docker client.APIClient
}

func (c *Collector) Collect(ctx context.Context, containerName string) (<-chan Stat, <-chan error) {
	stream := make(chan Stat, 1)
	errchan := make(chan error, 1)

	go func() {
		defer close(errchan)
		defer close(stream)

		res, err := c.Docker.ContainerStats(ctx, containerName, true)
		if err != nil {
			errchan <- errors.Wrap(err, "requesting container stats")
			return
		}

		defer res.Body.Close()

		decoder := json.NewDecoder(bufio.NewReader(res.Body))

		var stat container.StatsResponse
		for {
			if err := decoder.Decode(&stat); err != nil {
				if !errors.Is(err, context.Canceled) {
					errchan <- errors.Wrap(err, "decoding container stats")
				}
				return
			}

			result := Stat{Container: containerName}

			// used_memory = memory_stats.usage - memory_stats.stats.cache
			// available_memory = memory_stats.limit
			// memory usage = used_memory / available_memory
			result.MemoryUsage = float64(stat.MemoryStats.Usage-stat.MemoryStats.Stats["cache"]) / float64(stat.MemoryStats.Limit)

			// cpu_delta = cpu_stats.cpu_usage.total_usage - precpu_stats.cpu_usage.total_usage
			// system_cpu_delta = cpu_stats.system_cpu_usage - precpu_stats.system_cpu_usage
			// cpu usage = cpu_delta / system_cpu_delta
			cpuDelta := stat.CPUStats.CPUUsage.TotalUsage - stat.PreCPUStats.CPUUsage.TotalUsage
			systemDelta := stat.CPUStats.SystemUsage - stat.PreCPUStats.SystemUsage
			result.TotalCPUUsage = float64(cpuDelta) / float64(systemDelta)

			// Not sure if this approach is right.
			result.CPUUsagePerCore = make([]float64, len(stat.CPUStats.CPUUsage.PercpuUsage))
			for idx := range result.CPUUsagePerCore {
				// TODO: Change this into constant.
				const defaultCPUPeriod = float64(100_000)
				result.CPUUsagePerCore[idx] = float64(stat.CPUStats.CPUUsage.PercpuUsage[idx]) / defaultCPUPeriod
			}

			for _, networkStat := range stat.Networks {
				result.NetRead += networkStat.RxBytes
				result.NetWrite += networkStat.TxBytes
			}

			for _, blockStat := range stat.BlkioStats.IoServiceBytesRecursive {
				switch blockStat.Op {
				case "Read":
					result.BlockRead += blockStat.Value
				case "Write":
					result.BlockWrite += blockStat.Value
				}
			}

			stream <- result
		}
	}()

	return stream, errchan
}
