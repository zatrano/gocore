package database

import (
	"testing"
	"testing/fstest"
)

func TestLoadMigrations_pairsUpDown(t *testing.T) {
	fsys := fstest.MapFS{
		"000002_b.up.sql":   {Data: []byte("CREATE TABLE b();")},
		"000002_b.down.sql": {Data: []byte("DROP TABLE b;")},
		"000001_a.up.sql":   {Data: []byte("CREATE TABLE a();")},
		"000001_a.down.sql": {Data: []byte("DROP TABLE a;")},
		"readme.txt":        {Data: []byte("ignore")},
	}
	got, err := loadMigrations(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Version != 1 || got[1].Version != 2 {
		t.Fatalf("order: %+v", got)
	}
	if got[0].UpSQL == "" || got[0].DownSQL == "" {
		t.Fatal("expected up/down bodies")
	}
}

func TestPreviousVersion(t *testing.T) {
	files := []migrationFile{{Version: 1}, {Version: 3}, {Version: 5}}
	if previousVersion(files, 5) != 3 {
		t.Fatal("expected 3")
	}
	if previousVersion(files, 1) != 0 {
		t.Fatal("expected 0")
	}
}
