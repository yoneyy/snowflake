# snowflake

snowflake is a GO package

```
+--------------------------------------------------------------------------+
| 1 Bit Unused | 41 Bit Timestamp |  10 Bit NodeID  |   12 Bit Sequence ID |
+--------------------------------------------------------------------------+
```

# Get Started

## Installing

```sh
  go get github.com/yoneyy/snowflake
```

# Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/yoneyy/snowflake"
)

func main() {
	epoch := time.Date(2024, 9, 10, 15, 17, 0, 0, time.Local).UnixNano() / int64(time.Millisecond)
	id, err := snowflake.NewSnowflake(&snowflake.SnowflakeOption{
		NodeID: 1,
		Epoch:  epoch,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("id.ID() ==> %v\n", id.ID())
	fmt.Printf("id.ID().Int64() ==> %d\n", id.ID().Int64())
	fmt.Printf("id.ID().Bytes() ==> %v\n", id.ID().Bytes())
	fmt.Printf("id.ID().Base64() ==> %v\n", id.ID().Base64())
}
```