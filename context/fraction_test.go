package context

import (
	"context"
	"testing"
	"time"
)

func TestFractions2Deadlines(t *testing.T) {
	now, _ := time.Parse("2006-01-02 15:04:05.000", "2006-01-02 15:04:05.000")
	deadline := now.Add(10 * time.Second)
	d := fractionsToDeadlines(now, deadline, []float64{0.5, 0.5})
	if len(d) != 2 {
		t.Error("len(d) != 2")
		t.FailNow()
	}
	if d[0] != now.Add(5*time.Second) {
		t.Errorf("d[0] != now, want %v, got %v", now.Add(5*time.Second), d[0])
		t.FailNow()
	}
	if d[1] != deadline {
		t.Errorf("d[1] != deadline, want %v, got %v", deadline, d[1])
		t.FailNow()
	}
}

func TestFractionalContext(t *testing.T) {
	now := time.Now()
	t.Log("now", now.Format("2006-01-02 15:04:05.000"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx1, cancel1 := WithFractions(ctx, 0.4, 0.3)
	defer cancel1()
	d, _ := ctx1.Deadline()
	t.Log("chain context deadline", d.Format("2006-01-02 15:04:05.000"))

	c1 := ctx1.Next()
	d, _ = c1.Deadline()
	t.Log("c1 deadline", d.Format("2006-01-02 15:04:05.000"))
	t1 := <-time.After(3 * time.Second) // 3s
	select {
	case <-c1.Done():
		t.Error("c1 should be running, but now it's done at", t1.Format("2006-01-02 15:04:05.000"))
		t.FailNow()
	default:
	}

	c2 := ctx1.Next()
	d, _ = c2.Deadline()
	t.Log("c2 deadline", d.Format("2006-01-02 15:04:05.000"))
	select {
	case <-c1.Done():
	default:
		t.Error("c1 should be done, but now it's not after calling Next")
		t.FailNow()
	}
	t2 := <-time.After(5 * time.Second) // 3s + 5s
	select {
	case <-c2.Done():
	default:
		t.Error("c2 should be done, but now it's not at", t2.Format("2006-01-02 15:04:05.000"))
		t.FailNow()
	}

	t.Log("all done", time.Now().Format("2006-01-02 15:04:05.000"))
}
