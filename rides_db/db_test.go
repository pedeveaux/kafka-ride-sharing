package rides_db

import "testing"

func TestInit_BadConnectionString(t *testing.T) {
	err := Init("host=invalidhost user=bad password=bad dbname=none sslmode=disable")
	if err == nil {
		t.Error("Expected error from Init with bad connection string, got nil")
	}
}
