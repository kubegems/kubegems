package models

import (
	"context"
	"log"
	"testing"
)

func TestSyncService_SyncStatus(t *testing.T) {
	s := NewSyncService(&SyncOptions{
		Addr: "http://127.0.0.1:8000",
	})

	got, err := s.SyncStatus(context.Background(), "source")
	if err != nil {
		t.Errorf("SyncService.SyncStatus() error = %v", err)
		return
	}
	log.Print(got)
}
