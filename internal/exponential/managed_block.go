package exponential

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// ManagedBlock represents a delimited section of a file managed by xpo.
type ManagedBlock struct {
	Version string
	Hash    string
	Content string
	Start   int // byte offset of begin marker (inclusive)
	End     int // byte offset after end marker (exclusive)
}

var (
	mdBeginRe    = regexp.MustCompile(`<!-- xpo:begin(?:\s+(\S+))?(?:\s+sha256:([a-f0-9]+))? -->`)
	mdEndMarker  = "<!-- xpo:end -->"
	txtBeginRe   = regexp.MustCompile(`# xpo:begin(?:\s+(\S+))?(?:\s+sha256:([a-f0-9]+))?`)
	txtEndMarker = "# xpo:end"
)

// FindManagedBlock locates the xpo managed block in content.
// Format is "markdown" (HTML comments) or "plain" (# comments).
// Returns nil if no block is found.
func FindManagedBlock(content string, format string) *ManagedBlock {
	var beginRe *regexp.Regexp
	var endMarker string

	if format == "plain" {
		beginRe = txtBeginRe
		endMarker = txtEndMarker
	} else {
		beginRe = mdBeginRe
		endMarker = mdEndMarker
	}

	loc := beginRe.FindStringSubmatchIndex(content)
	if loc == nil {
		return nil
	}

	beginStart := loc[0]
	beginEnd := loc[1]

	var version, hash string
	if loc[2] != -1 {
		version = content[loc[2]:loc[3]]
	}
	if loc[4] != -1 {
		hash = content[loc[4]:loc[5]]
	}

	endIdx := strings.Index(content[beginEnd:], endMarker)
	if endIdx == -1 {
		return nil
	}
	endAbsolute := beginEnd + endIdx + len(endMarker)

	// Content is between the end of the begin marker line and the start of the end marker line
	innerStart := beginEnd
	if innerStart < len(content) && content[innerStart] == '\n' {
		innerStart++
	}
	innerEnd := beginEnd + endIdx
	if innerEnd > 0 && content[innerEnd-1] == '\n' {
		innerEnd--
	}

	inner := ""
	if innerEnd > innerStart {
		inner = content[innerStart:innerEnd]
	}

	return &ManagedBlock{
		Version: version,
		Hash:    hash,
		Content: inner,
		Start:   beginStart,
		End:     endAbsolute,
	}
}

// ComputeBlockHash returns the first 12 hex chars of a SHA-256 of the content.
func ComputeBlockHash(content string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return fmt.Sprintf("%x", h)[:12]
}

// BlockWasEdited returns true when the stored hash does not match the current content.
// Returns false if no hash was stored (e.g. legacy or first-time migration).
func BlockWasEdited(block *ManagedBlock) bool {
	if block == nil || block.Hash == "" {
		return false
	}
	return block.Hash != ComputeBlockHash(block.Content)
}

// WrapManagedBlock wraps content in managed block markers with version and hash.
func WrapManagedBlock(content string, version string, format string) string {
	hash := ComputeBlockHash(content)

	var begin, end string
	if format == "plain" {
		begin = fmt.Sprintf("# xpo:begin %s sha256:%s", version, hash)
		end = "# xpo:end"
	} else {
		begin = fmt.Sprintf("<!-- xpo:begin %s sha256:%s -->", version, hash)
		end = "<!-- xpo:end -->"
	}

	return begin + "\n" + content + "\n" + end
}

// ReplaceManagedBlock replaces an existing managed block in fileContent with new content.
func ReplaceManagedBlock(fileContent string, block *ManagedBlock, newContent string, version string, format string) string {
	wrapped := WrapManagedBlock(newContent, version, format)
	return fileContent[:block.Start] + wrapped + fileContent[block.End:]
}
