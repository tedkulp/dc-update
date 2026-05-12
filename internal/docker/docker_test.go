package docker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/docker/docker/api/types"
	imagetypes "github.com/docker/docker/api/types/image"
)

// fakeDockerAPI is a thread-safe in-process implementation of dockerAPI for testing.
type fakeDockerAPI struct {
	mu             sync.Mutex
	images         []imagetypes.Summary
	containers     map[string]types.ContainerJSON
	imageListCalls atomic.Int64
	inspectCalls   sync.Map // map[string]*atomic.Int64
}

func (f *fakeDockerAPI) ImageList(_ context.Context, _ imagetypes.ListOptions) ([]imagetypes.Summary, error) {
	f.imageListCalls.Add(1)
	f.mu.Lock()
	images := make([]imagetypes.Summary, len(f.images))
	copy(images, f.images)
	f.mu.Unlock()
	return images, nil
}

func (f *fakeDockerAPI) ContainerInspect(_ context.Context, containerID string) (types.ContainerJSON, error) {
	counter, _ := f.inspectCalls.LoadOrStore(containerID, new(atomic.Int64))
	counter.(*atomic.Int64).Add(1)

	f.mu.Lock()
	c, ok := f.containers[containerID]
	f.mu.Unlock()
	if !ok {
		return types.ContainerJSON{}, fmt.Errorf("No such container: %s", containerID)
	}
	return c, nil
}

func (f *fakeDockerAPI) Close() error { return nil }

func (f *fakeDockerAPI) inspectCallCount(containerID string) int64 {
	v, ok := f.inspectCalls.Load(containerID)
	if !ok {
		return 0
	}
	return v.(*atomic.Int64).Load()
}

// newTestClient builds a Client backed by a fakeDockerAPI, bypassing Docker daemon.
func newTestClient(api *fakeDockerAPI) *Client {
	return &Client{
		cli:            api,
		ctx:            context.Background(),
		imageCache:     make(map[string]*imagetypes.Summary),
		containerCache: make(map[string]*types.ContainerJSON),
	}
}

// makeImages builds a slice of imagetypes.Summary with the given repo tags.
func makeImages(tags ...string) []imagetypes.Summary {
	imgs := make([]imagetypes.Summary, 0, len(tags))
	for _, tag := range tags {
		imgs = append(imgs, imagetypes.Summary{
			ID:       "sha256:" + tag + "deadbeef",
			RepoTags: []string{tag},
		})
	}
	return imgs
}

const goroutines = 50

// TestGetImageIdConcurrent verifies that concurrent GetImageId calls are race-free
// and that ImageList is called exactly once despite the concurrency.
func TestGetImageIdConcurrent(t *testing.T) {
	fake := &fakeDockerAPI{
		images: makeImages("nginx:latest", "redis:alpine"),
	}
	c := newTestClient(fake)

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := c.GetImageId("nginx:latest")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if id == "" {
				t.Error("expected non-empty image ID")
			}
		}()
	}
	wg.Wait()

	if n := fake.imageListCalls.Load(); n != 1 {
		t.Errorf("ImageList called %d times, want exactly 1", n)
	}
}

// TestGetCurrentImageIdConcurrent verifies that concurrent GetCurrentImageId calls
// are race-free and that ContainerInspect is called exactly once per container.
func TestGetCurrentImageIdConcurrent(t *testing.T) {
	containerID := "abc123"
	fake := &fakeDockerAPI{
		containers: map[string]types.ContainerJSON{
			containerID: {
				ContainerJSONBase: &types.ContainerJSONBase{
					ID:    containerID,
					Image: "sha256:nginxdeadbeef",
				},
			},
		},
	}
	c := newTestClient(fake)

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := c.GetCurrentImageId(containerID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if id == "" {
				t.Error("expected non-empty image ID")
			}
		}()
	}
	wg.Wait()

	if n := fake.inspectCallCount(containerID); n != 1 {
		t.Errorf("ContainerInspect called %d times, want exactly 1", n)
	}
}

// TestRefreshImageCacheConcurrent verifies that concurrent GetImageId and
// RefreshImageCache calls are race-free and that the cache reflects the
// post-refresh state when all goroutines complete.
func TestRefreshImageCacheConcurrent(t *testing.T) {
	fake := &fakeDockerAPI{
		images: makeImages("nginx:latest"),
	}
	c := newTestClient(fake)

	var wg sync.WaitGroup

	// Half the goroutines read; half refresh.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, err := c.GetImageId("nginx:latest")
				if err != nil {
					t.Errorf("GetImageId error: %v", err)
				}
			} else {
				if err := c.RefreshImageCache(); err != nil {
					t.Errorf("RefreshImageCache error: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()

	// After all goroutines finish, cache should still return the image.
	id, err := c.GetImageId("nginx:latest")
	if err != nil {
		t.Fatalf("final GetImageId error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty image ID after concurrent refresh")
	}
}

// TestRefreshClearsContainerCache verifies that RefreshImageCache also invalidates
// the container cache so stale ContainerJSON entries don't survive a docker pull.
func TestRefreshClearsContainerCache(t *testing.T) {
	containerID := "abc123"
	fake := &fakeDockerAPI{
		images: makeImages("nginx:latest"),
		containers: map[string]types.ContainerJSON{
			containerID: {
				ContainerJSONBase: &types.ContainerJSONBase{
					ID:    containerID,
					Image: "sha256:oldimage",
				},
			},
		},
	}
	c := newTestClient(fake)

	// Populate container cache with old image.
	if _, err := c.GetCurrentImageId(containerID); err != nil {
		t.Fatalf("setup GetCurrentImageId: %v", err)
	}
	if n := fake.inspectCallCount(containerID); n != 1 {
		t.Fatalf("expected 1 inspect call after first lookup, got %d", n)
	}

	// Simulate a docker pull: update the fake to return a new image.
	fake.mu.Lock()
	fake.containers[containerID] = types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    containerID,
			Image: "sha256:newimage",
		},
	}
	fake.mu.Unlock()

	// Refresh should clear both caches.
	if err := c.RefreshImageCache(); err != nil {
		t.Fatalf("RefreshImageCache: %v", err)
	}

	// Next call must hit the API again (cache was cleared) and return the new image.
	id, err := c.GetCurrentImageId(containerID)
	if err != nil {
		t.Fatalf("post-refresh GetCurrentImageId: %v", err)
	}
	if id != "newimage" {
		t.Errorf("got image ID %q, want %q", id, "newimage")
	}
	if n := fake.inspectCallCount(containerID); n != 2 {
		t.Errorf("expected 2 inspect calls total, got %d", n)
	}
}
