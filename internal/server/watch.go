package server

// WatchDB is a no-op with ref-based storage. Real-time updates are handled
// by WatchGitRefs which watches .git/refs/xpo/ for ref changes.
func (s *Server) WatchDB() {}
