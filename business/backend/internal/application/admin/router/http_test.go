package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

type publicObjectStub struct {
	item storagemodel.Object
	err  error
	key  string
}

func (s *publicObjectStub) StorageDrivers(context.Context) []storagemodel.DriverDefinition {
	return nil
}
func (s *publicObjectStub) ListStorageChannels(context.Context, storagemodel.ChannelFilter) ([]storagemodel.Channel, error) {
	return nil, nil
}
func (s *publicObjectStub) PutStorageChannel(context.Context, appmodel.PutStorageChannel) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *publicObjectStub) SetStorageChannelEnabled(context.Context, appmodel.SetStorageEnabled) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *publicObjectStub) SetStorageDefault(context.Context, appmodel.SetStorageDefault) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *publicObjectStub) RetireStorageChannel(context.Context, appmodel.RetireStorage) (storagemodel.Channel, error) {
	return storagemodel.Channel{}, nil
}
func (s *publicObjectStub) TestStorageChannel(context.Context, appmodel.TestStorageChannel) (storagemodel.TestResult, error) {
	return storagemodel.TestResult{}, nil
}
func (s *publicObjectStub) GetStorageObject(_ context.Context, key string) (storagemodel.Object, error) {
	s.key = key
	return s.item, s.err
}

var _ service.Storage = (*publicObjectStub)(nil)

func TestServePublicUploadWritesObjectBytes(t *testing.T) {
	stub := &publicObjectStub{item: storagemodel.Object{Key: "_storage_test/ping.txt", ContentType: "text/plain; charset=utf-8", Content: "liveshop storage connectivity test\n"}}
	engine := ghttp.GetServer(t.Name())
	engine.SetAddr("127.0.0.1:0")
	engine.SetDumpRouterMap(false)
	engine.Group("/uploads", func(group *ghttp.RouterGroup) {
		group.GET("/:folder/:name", servePublicUpload(stub))
	})
	go func() { _ = engine.Start() }()
	t.Cleanup(func() { _ = engine.Shutdown() })
	deadline := time.Now().Add(2 * time.Second)
	for engine.GetListenedPort() <= 0 {
		if time.Now().After(deadline) {
			t.Fatal("server did not listen")
		}
		time.Sleep(10 * time.Millisecond)
	}
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/uploads/_storage_test/ping.txt", engine.GetListenedPort()))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || string(body) != stub.item.Content {
		t.Fatalf("status=%d body=%q key=%s", response.StatusCode, body, stub.key)
	}
	if stub.key != "_storage_test/ping.txt" {
		t.Fatalf("key=%s", stub.key)
	}
	if response.Header.Get("Content-Type") != stub.item.ContentType {
		t.Fatalf("content-type=%s", response.Header.Get("Content-Type"))
	}
}
