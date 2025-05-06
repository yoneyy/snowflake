package snowflake

import (
	"encoding/base64"
	"strconv"
)

type SnowflakeID int64

// Int64 returns an int64 of the snowflake ID
// @author yoneyy (y.tianyuan)
func (id SnowflakeID) Int64() int64 {
	return int64(id)
}

// String returns a string of the snowflake ID
// @author yoneyy (y.tianyuan)
func (id SnowflakeID) String() string {
	return strconv.FormatInt(id.Int64(), 10)
}

// Bytes returns a byte slice of the snowflake ID
// @author yoneyy (y.tianyuan)
func (id SnowflakeID) Bytes() []byte {
	return []byte(id.String())
}

// Base64 returns a base64 string of the snowflake ID
// @author yoneyy (y.tianyuan)
func (id SnowflakeID) Base64() string {
	return base64.StdEncoding.EncodeToString(id.Bytes())
}
