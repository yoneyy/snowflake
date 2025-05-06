package snowflake_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/yoneyy/snowflake"
)

func TestSnowflake(t *testing.T) {
	e := int64(1288834974657)
	now := time.Now()
	u := time.Unix(e/1000, (e%1000)*1000000)
	s := u.Sub(now)
	epoch := now.Add(s)
	t.Logf("((e  1000) * 1000000): %d\n", (e%1000)*1000000)
	t.Logf("u: %v\n", u)
	t.Logf("s: %v\n", s)
	t.Logf("epoch: %v\n", epoch)
	snow, err := snowflake.NewSnowflake(&snowflake.SnowflakeOption{
		NodeID: 1,
		// Epoch: ,
	})

	if err != nil {
		t.Error(err)
	}

	total := 10000

	wg := sync.WaitGroup{}
	wg.Add(total)

	for range total {
		go func() {
			defer wg.Done()
			id := snow.ID()
			fmt.Printf("id: %v\n", id)
		}()
	}

	wg.Wait()

	// t.Log(1 << 41)
	// t.Log((int64(-1) ^ (int64(-1) << 41)) / (365 * 24 * 3600 * 1000))
}

// 0000000000000000000000011111111111111111111111111111111111111111
