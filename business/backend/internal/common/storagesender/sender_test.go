package storagesender

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

func TestLocalPutWritesProbeFile(t *testing.T) {
	root := t.TempDir()
	sender := &Sender{root: root}
	url, err := sender.Put(context.Background(), storagemodel.DriverLocal, nil, "_storage_test/ping.txt", []byte("liveshop storage connectivity test\n"))
	if err != nil {
		t.Fatal(err)
	}
	if url != "/uploads/_storage_test/ping.txt" {
		t.Fatalf("url=%s", url)
	}
	object, err := sender.GetLocal("_storage_test/ping.txt")
	if err != nil || object.Content != "liveshop storage connectivity test\n" {
		t.Fatalf("get=%#v err=%v", object, err)
	}
	got, err := os.ReadFile(filepath.Join(root, "_storage_test", "ping.txt"))
	if err != nil || string(got) != "liveshop storage connectivity test\n" {
		t.Fatalf("file=%q err=%v", got, err)
	}
}

func TestLocalPutRejectsTraversal(t *testing.T) {
	sender := &Sender{root: t.TempDir()}
	if _, err := sender.Put(context.Background(), storagemodel.DriverLocal, nil, "../escape.txt", []byte("x")); err != storagemodel.ErrInvalid {
		t.Fatalf("got %v", err)
	}
}
