package tests

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// TestDockerAvailability checks if Docker is available and properly configured
func TestDockerAvailability(t *testing.T) {
	ctx := context.Background()

	// Try to get Docker info
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Logf("❌ Docker provider creation failed: %v", err)
		t.Logf("🔧 Possible solutions:")
		t.Logf("   1. Ensure Docker Desktop is running")
		t.Logf("   2. Check Docker Desktop settings -> General -> 'Expose daemon on tcp://localhost:2375 without TLS'")
		t.Logf("   3. Restart Docker Desktop")
		t.Logf("   4. Run 'docker version' in terminal to verify Docker is working")
		t.Skip("Docker is not available - skipping container tests")
		return
	}
	defer provider.Close()

	// Try to get Docker client info
	info, err := provider.Client().Info(ctx)
	if err != nil {
		t.Logf("❌ Failed to get Docker info: %v", err)
		t.Skip("Docker client not working - skipping container tests")
		return
	}

	t.Logf("✅ Docker is available!")
	t.Logf("📊 Docker Info:")
	t.Logf("   - Server Version: %s", info.ServerVersion)
	t.Logf("   - Operating System: %s", info.OperatingSystem)
	t.Logf("   - Total Memory: %d MB", info.MemTotal/(1024*1024))
	t.Logf("   - Docker Root Dir: %s", info.DockerRootDir)
}
