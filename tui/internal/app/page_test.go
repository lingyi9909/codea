package app

import "testing"

func TestPageChatIsZeroValue(t *testing.T) {
	if PageChat != Page(0) {
		t.Errorf("PageChat = %d, want 0 (zero value)", PageChat)
	}
}

func TestPageString(t *testing.T) {
	cases := []struct {
		page Page
		want string
	}{
		{PageChat, "chat"},
	}
	for _, c := range cases {
		if got := c.page.String(); got != c.want {
			t.Errorf("Page(%d).String() = %q, want %q", c.page, got, c.want)
		}
	}
}
