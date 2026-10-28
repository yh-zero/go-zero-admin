// Package iplocation resolves addresses using the bundled offline ip2region
// databases. It performs no network requests, including during construction.
package iplocation

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

type database struct {
	version *xdb.Version
	content []byte
}

// Resolver holds immutable database buffers and is safe for concurrent use.
// It retains no open files and does not require Close.
type Resolver struct {
	v4, v6 database
}

// New loads both databases. Missing, truncated or incompatible files are
// configuration errors; callers should report the returned error at startup.
func New(v4Path, v6Path string) (*Resolver, error) {
	v4, err := loadDatabase(v4Path, xdb.IPv4)
	if err != nil {
		return nil, err
	}
	v6, err := loadDatabase(v6Path, xdb.IPv6)
	if err != nil {
		return nil, err
	}
	return &Resolver{v4: v4, v6: v6}, nil
}

func loadDatabase(path string, version *xdb.Version) (database, error) {
	fail := func(err error) (database, error) {
		return database{}, fmt.Errorf("load %s IP location database %q: %w", version.Name, path, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return fail(err)
	}
	defer file.Close()
	if err = xdb.Verify(file); err != nil {
		return fail(err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return fail(err)
	}
	if err = validateBuffer(content, version); err != nil {
		return fail(err)
	}
	return database{version: version, content: content}, nil
}

// The upstream searcher slices the buffer using database pointers. Validate
// bounds once when loading so a truncated deployment file fails at startup.
func validateBuffer(content []byte, version *xdb.Version) error {
	const vectorEnd = xdb.HeaderInfoLength + xdb.VectorIndexRows*xdb.VectorIndexCols*xdb.VectorIndexSize
	if len(content) < vectorEnd {
		return fmt.Errorf("incomplete header/vector index")
	}
	header, err := xdb.NewHeader(content)
	if err != nil {
		return err
	}
	actualVersion, err := xdb.VersionFromHeader(header)
	if err != nil {
		return err
	}
	if actualVersion.Id != version.Id {
		return fmt.Errorf("expected %s, found %s", version.Name, actualVersion.Name)
	}
	if header.IndexPolicy != xdb.VectorIndexPolicy || (header.Version == xdb.Structure30 && header.RuntimePtrBytes != 4) {
		return fmt.Errorf("unsupported index policy or pointer width")
	}
	start, end, size := uint64(header.StartIndexPtr), uint64(header.EndIndexPtr), uint64(version.SegmentIndexSize)
	if start < vectorEnd || end < start || end+size > uint64(len(content)) || (end-start)%size != 0 {
		return fmt.Errorf("invalid segment index bounds")
	}
	for offset := xdb.HeaderInfoLength; offset < vectorEnd; offset += xdb.VectorIndexSize {
		s := uint64(binary.LittleEndian.Uint32(content[offset:]))
		e := uint64(binary.LittleEndian.Uint32(content[offset+4:]))
		if s == 0 && e == 0 {
			continue
		}
		// The final vector in the official files ends one segment past the
		// header's last index; the reader handles that boundary as a short read.
		if s < start || s > end || e < s || e > end+size || (s-start)%size != 0 || (e-start)%size != 0 {
			return fmt.Errorf("invalid vector index at %d", offset)
		}
	}
	for offset := start; offset <= end; offset += size {
		dataOffset := offset + uint64(2*version.Bytes)
		length := uint64(binary.LittleEndian.Uint16(content[dataOffset:]))
		pointer := uint64(binary.LittleEndian.Uint32(content[dataOffset+2:]))
		if pointer+length > uint64(len(content)) {
			return fmt.Errorf("invalid region data at %d", offset)
		}
	}
	return nil
}

var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

// Lookup accepts a bare IPv4/IPv6 address, including IPv4-mapped IPv6.
// Private, shared-address-space, loopback and link-local unicast IPs return
// 内网; invalid, non-unicast and unmatched addresses return 未知.
func (r *Resolver) Lookup(raw string) string {
	ip, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil || ip.Zone() != "" {
		return "未知"
	}
	ip = ip.Unmap()
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || sharedAddressSpace.Contains(ip) {
		return "内网"
	}
	if r == nil || !ip.IsGlobalUnicast() {
		return "未知"
	}
	db := &r.v6
	if ip.Is4() {
		db = &r.v4
	}
	if len(db.content) == 0 {
		return "未知"
	}
	// Upstream Search writes an internal I/O counter even in memory mode.
	// Each call owns a lightweight searcher while sharing only immutable bytes.
	searcher, err := xdb.NewWithBuffer(db.version, db.content)
	if err != nil {
		return "未知"
	}
	region, err := searcher.Search(ip.AsSlice())
	if err != nil {
		return "未知"
	}
	return displayRegion(region)
}

func displayRegion(region string) string {
	fields := strings.Split(region, "|")
	parts := make([]string, 0, 3)
	seen := make(map[string]bool, 3)
	// The pinned data format is Country|Province|City|ISP|ISO-alpha2.
	for i := 0; i < len(fields) && i < 3; i++ {
		value := strings.TrimSpace(fields[i])
		if value == "" || value == "0" || seen[value] {
			continue
		}
		seen[value] = true
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "未知"
	}
	return strings.Join(parts, " / ")
}
