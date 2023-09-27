/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapshots

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/webmeshproj/webmesh/pkg/storage/backends/badgerdb"
)

func TestSnapshotter(t *testing.T) {
	t.Parallel()

	db, err := badgerdb.NewInMemory(badgerdb.Options{})
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	defer db.Close()

	// Create a test table and populate it with some data.
	testValues := map[string][]byte{
		"/registry/foo": []byte("bar"),
		"/registry/baz": []byte("qux"),
		"/registry/abc": []byte("def"),
	}
	for key, val := range testValues {
		if err := db.PutValue(context.Background(), []byte(key), val, 0); err != nil {
			t.Fatal(err)
		}
	}
	snaps := New(context.Background(), db)

	// Take a snapshot.
	snap, err := snaps.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Drop the keys
	for key := range testValues {
		if err := db.Delete(context.Background(), []byte(key)); err != nil {
			t.Fatal(err)
		}
	}

	// Persist the snapshot to a buffer.
	buf := new(bytes.Buffer)
	sink := &testSnapshotSink{buf}
	if err := snap.Persist(sink); err != nil {
		t.Fatal(err)
	}

	// Restore the snapshot.
	if err := snaps.Restore(context.Background(), sink); err != nil {
		t.Fatal(err)
	}

	snap.Release()

	// Ensure the keys were restored.
	for key, val := range testValues {
		got, err := db.GetValue(context.Background(), []byte(key))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, val) {
			t.Errorf("got %q, want %q", got, val)
		}
	}
}

type testSnapshotSink struct {
	io.ReadWriter
}

func (t *testSnapshotSink) ID() string {
	return "test"
}

func (t *testSnapshotSink) Cancel() error {
	return nil
}

func (t *testSnapshotSink) Close() error {
	return nil
}
