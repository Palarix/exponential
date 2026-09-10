package exponential

import (
	"strings"
	"testing"
)

func TestFindManagedBlock_Markdown(t *testing.T) {
	content := `# My Project

Custom docs here.

<!-- xpo:begin 1.2.1 sha256:abc123def456 -->
# Agent Instructions

Some managed content.
<!-- xpo:end -->

More user content below.
`
	block := FindManagedBlock(content, "markdown")
	if block == nil {
		t.Fatal("expected to find managed block")
	}
	if block.Version != "1.2.1" {
		t.Fatalf("expected version 1.2.1, got %s", block.Version)
	}
	if block.Hash != "abc123def456" {
		t.Fatalf("expected hash abc123def456, got %s", block.Hash)
	}
	if !strings.Contains(block.Content, "# Agent Instructions") {
		t.Fatal("content should contain the heading")
	}
	if !strings.Contains(block.Content, "Some managed content.") {
		t.Fatal("content should contain the body")
	}
	// Verify offsets don't include surrounding content
	before := content[:block.Start]
	if !strings.Contains(before, "Custom docs here.") {
		t.Fatal("content before block should include user content")
	}
	after := content[block.End:]
	if !strings.Contains(after, "More user content below.") {
		t.Fatal("content after block should include user content")
	}
}

func TestFindManagedBlock_Plain(t *testing.T) {
	content := `Some preamble.

# xpo:begin 1.0.0 sha256:aabbccdd1122
Agent instructions for plain text.
# xpo:end

Footer text.
`
	block := FindManagedBlock(content, "plain")
	if block == nil {
		t.Fatal("expected to find managed block in plain format")
	}
	if block.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", block.Version)
	}
	if !strings.Contains(block.Content, "Agent instructions for plain text.") {
		t.Fatal("content mismatch")
	}
}

func TestFindManagedBlock_NoBlock(t *testing.T) {
	content := "# My Project\n\nNo managed block here.\n"
	block := FindManagedBlock(content, "markdown")
	if block != nil {
		t.Fatal("expected nil for content without managed block")
	}
}

func TestFindManagedBlock_NoEndMarker(t *testing.T) {
	content := "<!-- xpo:begin 1.0.0 sha256:abc123 -->\nOrphan block\n"
	block := FindManagedBlock(content, "markdown")
	if block != nil {
		t.Fatal("expected nil when end marker is missing")
	}
}

func TestFindManagedBlock_NoVersionOrHash(t *testing.T) {
	content := "<!-- xpo:begin -->\nContent\n<!-- xpo:end -->\n"
	block := FindManagedBlock(content, "markdown")
	if block == nil {
		t.Fatal("expected to find block without version/hash")
	}
	if block.Version != "" {
		t.Fatalf("expected empty version, got %s", block.Version)
	}
	if block.Hash != "" {
		t.Fatalf("expected empty hash, got %s", block.Hash)
	}
}

func TestComputeBlockHash_Deterministic(t *testing.T) {
	h1 := ComputeBlockHash("hello world")
	h2 := ComputeBlockHash("hello world")
	if h1 != h2 {
		t.Fatal("hash should be deterministic")
	}
	if len(h1) != 12 {
		t.Fatalf("expected 12 char hash, got %d", len(h1))
	}
}

func TestComputeBlockHash_TrimWhitespace(t *testing.T) {
	h1 := ComputeBlockHash("hello")
	h2 := ComputeBlockHash("  hello  \n")
	if h1 != h2 {
		t.Fatal("hash should be whitespace-insensitive")
	}
}

func TestBlockWasEdited_UnchangedContent(t *testing.T) {
	content := "# Agent Instructions\n\nSome content."
	hash := ComputeBlockHash(content)
	block := &ManagedBlock{
		Version: "1.0.0",
		Hash:    hash,
		Content: content,
	}
	if BlockWasEdited(block) {
		t.Fatal("block should not be detected as edited when hash matches")
	}
}

func TestBlockWasEdited_ChangedContent(t *testing.T) {
	original := "# Agent Instructions\n\nOriginal content."
	hash := ComputeBlockHash(original)
	block := &ManagedBlock{
		Version: "1.0.0",
		Hash:    hash,
		Content: "# Agent Instructions\n\nEdited content.",
	}
	if !BlockWasEdited(block) {
		t.Fatal("block should be detected as edited when content changed")
	}
}

func TestBlockWasEdited_NoHash(t *testing.T) {
	block := &ManagedBlock{
		Version: "1.0.0",
		Hash:    "",
		Content: "anything",
	}
	if BlockWasEdited(block) {
		t.Fatal("block without hash should not be detected as edited")
	}
}

func TestBlockWasEdited_Nil(t *testing.T) {
	if BlockWasEdited(nil) {
		t.Fatal("nil block should not be detected as edited")
	}
}

func TestWrapManagedBlock_Markdown(t *testing.T) {
	content := "# Agent Instructions\n\nContent here."
	wrapped := WrapManagedBlock(content, "1.2.1", "markdown")

	if !strings.HasPrefix(wrapped, "<!-- xpo:begin 1.2.1 sha256:") {
		t.Fatal("wrapped content should start with markdown begin marker")
	}
	if !strings.HasSuffix(wrapped, "<!-- xpo:end -->") {
		t.Fatal("wrapped content should end with markdown end marker")
	}
	if !strings.Contains(wrapped, content) {
		t.Fatal("wrapped content should contain the original content")
	}

	// Should be parseable
	block := FindManagedBlock(wrapped, "markdown")
	if block == nil {
		t.Fatal("wrapped content should be parseable")
	}
	if block.Version != "1.2.1" {
		t.Fatalf("expected version 1.2.1, got %s", block.Version)
	}
	if BlockWasEdited(block) {
		t.Fatal("freshly wrapped block should not be detected as edited")
	}
}

func TestWrapManagedBlock_Plain(t *testing.T) {
	content := "Agent instructions."
	wrapped := WrapManagedBlock(content, "2.0.0", "plain")

	if !strings.HasPrefix(wrapped, "# xpo:begin 2.0.0 sha256:") {
		t.Fatal("wrapped content should start with plain begin marker")
	}
	if !strings.HasSuffix(wrapped, "# xpo:end") {
		t.Fatal("wrapped content should end with plain end marker")
	}
}

func TestReplaceManagedBlock(t *testing.T) {
	fileContent := `# My Project

Custom docs.

<!-- xpo:begin 1.0.0 sha256:aabbccddeeff -->
Old managed content.
<!-- xpo:end -->

Footer.
`
	block := FindManagedBlock(fileContent, "markdown")
	if block == nil {
		t.Fatal("setup: expected to find block")
	}

	result := ReplaceManagedBlock(fileContent, block, "New managed content.", "1.2.1", "markdown")

	// User content preserved
	if !strings.Contains(result, "# My Project") {
		t.Fatal("header lost")
	}
	if !strings.Contains(result, "Custom docs.") {
		t.Fatal("custom docs lost")
	}
	if !strings.Contains(result, "Footer.") {
		t.Fatal("footer lost")
	}

	// Old content gone
	if strings.Contains(result, "Old managed content") {
		t.Fatal("old content should be replaced")
	}
	if strings.Contains(result, "aabbccddeeff") {
		t.Fatal("old hash should be replaced")
	}

	// New block present and valid
	newBlock := FindManagedBlock(result, "markdown")
	if newBlock == nil {
		t.Fatal("new block not found")
	}
	if newBlock.Version != "1.2.1" {
		t.Fatalf("expected version 1.2.1, got %s", newBlock.Version)
	}
	if !strings.Contains(newBlock.Content, "New managed content.") {
		t.Fatal("new content not in block")
	}
	if BlockWasEdited(newBlock) {
		t.Fatal("freshly replaced block should not be edited")
	}
}
