package profile

import (
	"os"
	"testing"
)

func TestScenePresetManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex_scene_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewScenePresetManager(tempDir)

	// 1. Initial State
	status := mgr.GetStatus()
	if status.ActiveScene != SceneBalanced {
		t.Errorf("expected initial scene balanced, got %s", status.ActiveScene)
	}

	// 2. Switch to Gaming
	params, err := mgr.SwitchScene(SceneGaming)
	if err != nil {
		t.Fatalf("switch scene failed: %v", err)
	}
	if params.DSCPValue != 46 || params.CoDelTargetMs != 1.0 {
		t.Errorf("gaming params mismatch: %+v", params)
	}

	status = mgr.GetStatus()
	if status.ActiveScene != SceneGaming {
		t.Errorf("expected active scene gaming, got %s", status.ActiveScene)
	}

	// 3. Test Persistence: Create another instance from same dir
	mgr2 := NewScenePresetManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.ActiveScene != SceneGaming {
		t.Errorf("expected loaded scene gaming, got %s", status2.ActiveScene)
	}

	// 4. Invalid scene test
	_, err = mgr.SwitchScene("invalid_scene")
	if err == nil {
		t.Errorf("expected error switching to invalid scene, got nil")
	}
}
