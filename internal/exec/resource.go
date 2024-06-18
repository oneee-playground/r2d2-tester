package exec

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/job"
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

		natPort := nat.Port(fmt.Sprintf("%d/tcp", resource.Port))

		containerConf := &container.Config{
			Image:        resource.Image,
			Hostname:     resource.Name,
			Domainname:   resource.Name,
			ExposedPorts: nat.PortSet{natPort: struct{}{}},
			// Volumes:      map[string]struct{}{},
			// Healthcheck:  &container.HealthConfig{},
		}

		hostConf := &container.HostConfig{
			NetworkMode: network.NetworkBridge,
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

		r, err := e.Docker.ImagePull(ctx, resource.Image, image.PullOptions{Platform: "linux/amd64"})
		if err != nil {
			return errors.Wrap(err, "pulling image")
		}

		if _, err := io.Copy(io.Discard, r); err != nil {
			return errors.Wrap(err, "reading output from docker daemon")
		}

		res, err := e.Docker.ContainerCreate(ctx, containerConf, hostConf, nil, platformConf, resource.Name)
		if err != nil {
			return errors.Wrap(err, "creating container")
		}

		if len(res.Warnings) > 0 {
			e.Log.Warn("warning during container creation", zap.Strings("warnings", res.Warnings))
		}

		if err := e.Docker.ContainerStart(ctx, res.ID, container.StartOptions{}); err != nil {
			return errors.Wrap(err, "starting container")
		}

		*proc = process{
			ID:       res.ID,
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
