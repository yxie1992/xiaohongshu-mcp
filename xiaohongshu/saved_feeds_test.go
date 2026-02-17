package xiaohongshu

import "testing"

func TestTrimLeadingFeedsByAuthor(t *testing.T) {
	t.Parallel()

	newFeed := func(authorID, feedID string) Feed {
		return Feed{
			ID: feedID,
			NoteCard: NoteCard{
				User: User{UserID: authorID},
			},
		}
	}

	tests := []struct {
		name        string
		feeds       []Feed
		authorID    string
		wantFeedIDs []string
	}{
		{
			name:        "empty input",
			feeds:       []Feed{},
			authorID:    "u1",
			wantFeedIDs: []string{},
		},
		{
			name: "trim leading owner posts",
			feeds: []Feed{
				newFeed("owner", "f1"),
				newFeed("owner", "f2"),
				newFeed("u2", "f3"),
			},
			authorID:    "owner",
			wantFeedIDs: []string{"f3"},
		},
		{
			name: "owner post in middle does not trim",
			feeds: []Feed{
				newFeed("u2", "f1"),
				newFeed("owner", "f2"),
				newFeed("u3", "f3"),
			},
			authorID:    "owner",
			wantFeedIDs: []string{"f1", "f2", "f3"},
		},
		{
			name: "all owner posts keep original to avoid empty result",
			feeds: []Feed{
				newFeed("owner", "f1"),
				newFeed("owner", "f2"),
			},
			authorID:    "owner",
			wantFeedIDs: []string{"f1", "f2"},
		},
		{
			name: "blank author id keeps original",
			feeds: []Feed{
				newFeed("owner", "f1"),
				newFeed("u2", "f2"),
			},
			authorID:    "",
			wantFeedIDs: []string{"f1", "f2"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := trimLeadingFeedsByAuthor(tt.feeds, tt.authorID)
			if len(got) != len(tt.wantFeedIDs) {
				t.Fatalf("trimLeadingFeedsByAuthor got len=%d, want=%d", len(got), len(tt.wantFeedIDs))
			}
			for i := range got {
				if got[i].ID != tt.wantFeedIDs[i] {
					t.Fatalf("trimLeadingFeedsByAuthor item %d id=%s, want=%s", i, got[i].ID, tt.wantFeedIDs[i])
				}
			}
		})
	}
}

func TestDistinctAuthorCount(t *testing.T) {
	t.Parallel()

	feeds := []Feed{
		{NoteCard: NoteCard{User: User{UserID: "u1"}}},
		{NoteCard: NoteCard{User: User{UserID: "u2"}}},
		{NoteCard: NoteCard{User: User{UserID: "u2"}}},
		{NoteCard: NoteCard{User: User{UserID: ""}}},
	}

	got := distinctAuthorCount(feeds)
	if got != 2 {
		t.Fatalf("distinctAuthorCount() = %d, want 2", got)
	}
}
