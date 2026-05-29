package sortorder

import (
	"testing"
)

func TestGenerateKeyBetweenNilNil(t *testing.T) {
	key, err := GenerateKeyBetween("", "")
	if err != nil {
		t.Fatal(err)
	}
	if key != "a0" {
		t.Fatalf("expected a0, got %s", key)
	}
}

func TestGenerateKeyBetweenAfter(t *testing.T) {
	keys := []string{"a0"}
	for _, expected := range []string{"a1", "a2", "a3", "a4"} {
		key, err := GenerateKeyBetween(keys[len(keys)-1], "")
		if err != nil {
			t.Fatal(err)
		}
		if key != expected {
			t.Fatalf("expected %s, got %s", expected, key)
		}
		keys = append(keys, key)
	}
}

func TestGenerateKeyBetweenBefore(t *testing.T) {
	key, err := GenerateKeyBetween("", "a0")
	if err != nil {
		t.Fatal(err)
	}
	if key != "Zz" {
		t.Fatalf("expected Zz, got %s", key)
	}
}

func TestGenerateKeyBetweenMidpoint(t *testing.T) {
	key, err := GenerateKeyBetween("a0", "a2")
	if err != nil {
		t.Fatal(err)
	}
	if key != "a1" {
		t.Fatalf("expected a1, got %s", key)
	}
}

func TestGenerateKeyBetweenConsecutive(t *testing.T) {
	key, err := GenerateKeyBetween("a0", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if key <= "a0" || key >= "a1" {
		t.Fatalf("key %s is not between a0 and a1", key)
	}
}

func TestGenerateNKeysBetweenFromScratch(t *testing.T) {
	keys, err := GenerateNKeysBetween("", "", 5)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"a0", "a1", "a2", "a3", "a4"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}
	for i, k := range keys {
		if k != expected[i] {
			t.Fatalf("keys[%d]: expected %s, got %s", i, expected[i], k)
		}
	}
}

func TestGenerateNKeysBetweenOrdered(t *testing.T) {
	keys, err := GenerateNKeysBetween("a0", "a4", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	prev := "a0"
	for _, k := range keys {
		if k <= prev {
			t.Fatalf("key %s is not after %s", k, prev)
		}
		if k >= "a4" {
			t.Fatalf("key %s is not before a4", k)
		}
		prev = k
	}
}

func TestGenerateNKeysBetweenAfterExisting(t *testing.T) {
	keys, err := GenerateNKeysBetween("a4", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	prev := "a4"
	for _, k := range keys {
		if k <= prev {
			t.Fatalf("key %s is not after %s", k, prev)
		}
		prev = k
	}
}

func TestGenerateNKeysBetweenBeforeExisting(t *testing.T) {
	keys, err := GenerateNKeysBetween("", "a0", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Fatalf("keys not in order: %v", keys)
		}
	}
	if keys[len(keys)-1] >= "a0" {
		t.Fatalf("last key %s should be before a0", keys[len(keys)-1])
	}
}

func TestGenerateNKeysBetweenZero(t *testing.T) {
	keys, err := GenerateNKeysBetween("", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestIncrementWrapAround(t *testing.T) {
	key, err := GenerateKeyBetween("az", "")
	if err != nil {
		t.Fatal(err)
	}
	if key <= "az" {
		t.Fatalf("expected key after az, got %s", key)
	}
}

func TestKeysBetweenManyOperations(t *testing.T) {
	keys, err := GenerateNKeysBetween("", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Fatalf("keys not sorted at index %d: %s >= %s", i, keys[i-1], keys[i])
		}
	}
}
