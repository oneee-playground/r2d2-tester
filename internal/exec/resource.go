package exec

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/influxdata/influxdb-client-go/api/write"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/metric"
	"github.com/oneee-playground/r2d2-tester/internal/util/stream"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const defaultCPUPeriod = 100_000

func (e *Executor) setupResources(
	ctx context.Context, taskID uuid.UUID,
	resources []job.Resource, submission job.Submission,
) error {
	e.Log.Info("setting up resources")

	start := time.Now()

	e.processes = make([]*process, 0, len(resources))
	for _, resource := range resources {
		proc := new(process)

		if resource.IsPrimary {
			e.primaryProcess = proc

			// TODO: Remove hardcoded value.
			const username = "oneeonly"
			const registry = "docker.io"
			resource.Image = makeCustomImageName(
				registry, username, taskID,
				submission.Repository, submission.CommitHash,
			)
		}

		e.Log.Debug("resource info", zap.Any("info", resource))

		port := strconv.Itoa(int(resource.Port))
		natPort, err := nat.NewPort("tcp", port)
		if err != nil {
			return errors.Wrap(err, "parsing binding")
		}

		containerConf := &container.Config{
			Image:      resource.Image,
			Hostname:   resource.Name,
			Domainname: resource.Name,
			// Volumes:     map[string]struct{}{},
			// Healthcheck: &container.HealthConfig{},
			ExposedPorts: nat.PortSet{natPort: struct{}{}},
		}

		if resource.Name == "db" {
			containerConf.Env = append(containerConf.Env, "MYSQL_ALLOW_EMPTY_PASSWORD=true")
		}

		hostConf := &container.HostConfig{
			NetworkMode: container.NetworkMode(e.ExecNetwork),
			Resources: container.Resources{
				Memory:    int64(resource.Memory),
				CPUPeriod: defaultCPUPeriod,
				CPUQuota:  int64(resource.CPU * float64(defaultCPUPeriod)),
			},
		}

		platformConf := &v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		}

		content, err := e.Docker.ImagePull(ctx, resource.Image, image.PullOptions{Platform: "linux/amd64"})
		if err != nil {
			return errors.Wrap(err, "pulling image")
		}

		if _, err := io.Copy(io.Discard, content); err != nil {
			return errors.Wrap(err, "reading output from docker daemon")
		}

		con, err := e.Docker.ContainerCreate(ctx, containerConf, hostConf, nil, platformConf, resource.Name)
		if err != nil {
			return errors.Wrap(err, "creating container")
		}

		if len(con.Warnings) > 0 {
			e.Log.Warn("warning during container creation", zap.Strings("warnings", con.Warnings))
		}

		if resource.IsPrimary {
			// In order to send request to primary process from teseter,
			// primary process should be connected to the test network.
			e.Log.Info("resource is primary. connecting to test network")

			if err := e.Docker.NetworkConnect(ctx, e.TestNetwork, con.ID, nil); err != nil {
				return errors.Wrap(err, "connecting primary process to test network")
			}
		}

		if err := e.Docker.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
			return errors.Wrap(err, "starting container")
		}

		*proc = process{
			ID:       con.ID,
			Hostname: resource.Name,
			Port:     resource.Port,
			Image:    resource.Image,
		}
		e.processes = append(e.processes, proc)
	}

	e.Log.Info("resource setup done", zap.Duration("took", time.Since(start)))

	return nil
}

func (e *Executor) teardownResources(ctx context.Context) {
	e.Log.Info("tearing down resources")

	start := time.Now()

	for _, process := range e.processes {
		if err := e.Docker.ContainerStop(ctx, process.ID, container.StopOptions{}); err != nil {
			e.Log.Error("failed to stop container",
				zap.String("containerID", process.ID),
				zap.Error(err),
			)
		}
	}

	{
		report, err := e.Docker.ContainersPrune(ctx, filters.NewArgs())
		if err != nil {
			e.Log.Error("failed to prune containers", zap.Error(err))
			return
		}

		e.Log.Info("containers pruned",
			zap.Strings("containers", report.ContainersDeleted),
			zap.Uint64("space-reclaimed", report.SpaceReclaimed),
		)
	}

	{
		report, err := e.Docker.ImagesPrune(ctx, filters.NewArgs())
		if err != nil {
			e.Log.Error("failed to prune images", zap.Error(err))
			return
		}

		e.Log.Info("images pruned",
			zap.Any("images", report.ImagesDeleted),
			zap.Uint64("space-reclaimed", report.SpaceReclaimed),
		)
	}

	e.Log.Info("resource teardown done", zap.Duration("took", time.Since(start)))
}

func makeCustomImageName(registry, username string, taskID uuid.UUID, repository, commitHash string) string {
	return fmt.Sprintf("%s/%s/%s:%s-%s", registry, username, taskID.String(), strings.Replace(repository, "/", "-", 1), commitHash)
}

func (e *Executor) startMetricCollection(ctx context.Context, cancel func(error)) {
	collector := metric.Collector{Docker: e.Docker}

	statStreams := make([]<-chan metric.Stat, len(e.processes))
	errchans := make([]<-chan error, len(e.processes))

	for idx, proc := range e.processes {
		statStream, errchan := collector.Collect(ctx, proc.Hostname)
		statStreams[idx] = statStream
		errchans[idx] = errchan
	}

	go func() {
		statStream := stream.FanIn(statStreams...)
		errchan := stream.FanIn(errchans...)

		var stat metric.Stat
		var ok bool
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errchan:
				cancel(err)
				return
			case stat, ok = <-statStream:
				if !ok {
					return
				}
			}

			tags := map[string]string{"container": stat.Container}

			fields := map[string]interface{}{
				"cpu-usage":    stat.TotalCPUUsage,
				"memory-usage": stat.MemoryUsage,
				"block-read":   stat.BlockRead,
				"block-write":  stat.BlockWrite,
				"net-read":     stat.NetRead,
				"net-write":    stat.NetWrite,
			}

			for idx, v := range stat.CPUUsagePerCore {
				fields["cpu-usage-core-"+strconv.Itoa(idx)] = v
			}

			e.metrics.Write(write.NewPoint("resource-usage", tags, fields, time.Now()))
		}
	}()
}

func (e *Executor) setTimestamp(t time.Time, sectionID uuid.UUID, label string) {
	e.metrics.Write(write.NewPoint(
		"label",
		map[string]string{"section-id": sectionID.String()},
		map[string]interface{}{"label": label},
		t,
	))
}
