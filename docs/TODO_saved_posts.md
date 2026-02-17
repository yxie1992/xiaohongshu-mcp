# Saved Posts Feature TODO

## Goal
Add capability to access the current user's saved posts and interact with them.

## Phase 1 (This change)
- [x] Verify real headless workflow with logged-in cookies (`profile -> 收藏`).
- [x] Confirm saved-post metadata can be extracted from page state.
- [x] Implement `list saved posts` action in `xiaohongshu` package.
- [x] Add service method for listing saved posts.
- [x] Add HTTP API endpoint for listing saved posts.
- [x] Add MCP tool for listing saved posts.
- [x] Add unit tests for saved-post helper logic and limit normalization.
- [x] Update README/API docs with saved-post endpoint and tool.
- [x] Add PR CI workflow for `go vet` + `go test`.
- [x] Verify end-to-end in headless mode with real account cookies.

## Phase 2 (Next)
- [ ] Add safe interaction helpers for saved posts (e.g., like/unlike, favorite/unfavorite, comment) using returned `feed_id` and `xsec_token`.
- [ ] Add docs examples for saved-post workflow.
