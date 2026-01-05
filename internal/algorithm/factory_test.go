package algorithm

import (
	"testing"
	"time"

	"github.com/ASHISH26940/throttle/internal/types"
)

func TestFactoryCreate(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  20,
	}

	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}

	// Default should create token bucket
	alg, err := factory.Create(cfg)
	if err != nil {
		t.Fatalf("Factory.Create() failed: %v", err)
	}
	defer alg.Close()

	if alg.Type() != AlgorithmTokenBucket {
		t.Errorf("Expected default to be TokenBucket, got %v", alg.Type())
	}
}

func TestCreateByTypeTokenBucket(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  20,
	}

	alg, err := CreateByType(AlgorithmTokenBucket, cfg)
	if err != nil {
		t.Fatalf("CreateByType(TokenBucket) failed: %v", err)
	}
	defer alg.Close()

	if alg.Type() != AlgorithmTokenBucket {
		t.Errorf("Expected TokenBucket, got %v", alg.Type())
	}

	// Test basic functionality
	if err := alg.Allow("test"); err != nil {
		t.Errorf("Allow() failed: %v", err)
	}
}

func TestCreateByTypeLeakyBucket(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  20,
	}

	alg, err := CreateByType(AlgorithmLeakyBucket, cfg)
	if err != nil {
		t.Fatalf("CreateByType(LeakyBucket) failed: %v", err)
	}
	defer alg.Close()

	if alg.Type() != AlgorithmLeakyBucket {
		t.Errorf("Expected LeakyBucket, got %v", alg.Type())
	}

	// Test basic functionality
	if err := alg.Allow("test"); err != nil {
		t.Errorf("Allow() failed: %v", err)
	}
}

func TestCreateByTypeFixedWindow(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  0,
	}

	alg, err := CreateByType(AlgorithmFixedWindow, cfg)
	if err != nil {
		t.Fatalf("CreateByType(FixedWindow) failed: %v", err)
	}
	defer alg.Close()

	if alg.Type() != AlgorithmFixedWindow {
		t.Errorf("Expected FixedWindow, got %v", alg.Type())
	}

	// Test basic functionality
	if err := alg.Allow("test"); err != nil {
		t.Errorf("Allow() failed: %v", err)
	}
}

func TestCreateByTypeSlidingWindow(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  0,
	}

	alg, err := CreateByType(AlgorithmSlidingWindow, cfg)
	if err != nil {
		t.Fatalf("CreateByType(SlidingWindow) failed: %v", err)
	}
	defer alg.Close()

	if alg.Type() != AlgorithmSlidingWindow {
		t.Errorf("Expected SlidingWindow, got %v", alg.Type())
	}

	// Test basic functionality
	if err := alg.Allow("test"); err != nil {
		t.Errorf("Allow() failed: %v", err)
	}
}

func TestCreateByTypeInvalid(t *testing.T) {
	cfg := types.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  20,
	}

	alg, err := CreateByType(AlgorithmType(999), cfg)
	if err == nil {
		t.Error("Expected error for invalid algorithm type")
		if alg != nil {
			alg.Close()
		}
	}
}

func TestCreateByTypeInvalidConfig(t *testing.T) {
	cfg := types.Config{
		Rate:   -1, // Invalid
		Window: time.Second,
		Burst:  20,
	}

	alg, err := CreateByType(AlgorithmTokenBucket, cfg)
	if err == nil {
		t.Error("Expected error for invalid config")
		if alg != nil {
			alg.Close()
		}
	}
}
