package showdown

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestParseFormatsLine(t *testing.T) {
	line := "|formats|,1|Showdown Suite|[Gen 9] Showdown Suite Studio,4|,2|S/V Singles|[Gen 9] OU,4"
	want := []string{"[Gen 9] Showdown Suite Studio", "[Gen 9] OU"}

	got := parseFormatsLine(line)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFormatsLine() = %#v, want %#v", got, want)
	}
}

func TestRandomUsernameUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 512; i++ {
		name := randomUsername("mock")
		if seen[name] {
			t.Fatalf("randomUsername() generated duplicate %q", name)
		}
		seen[name] = true
	}
}

func TestQueryRejectsConcurrentSameType(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8000", "tester")
	if err != nil {
		t.Fatal(err)
	}
	client.queryWaiters["roomlist"] = make(chan QueryResponse, 1)

	_, err = client.Query(context.Background(), "roomlist")
	if err == nil {
		t.Fatal("Query() should reject duplicate in-flight query types")
	}
	if !strings.Contains(err.Error(), "already in flight") {
		t.Fatalf("Query() error = %q, want duplicate in-flight message", err.Error())
	}
}
