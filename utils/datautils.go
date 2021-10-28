package utils

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"math/big"
	"strconv"
	"time"
	"encoding/hex"
	config "xqledger/rdboperator/configuration"
)

var configuration = config.GlobalConfiguration

func GetRDBID() (string, error) {
	bytes := make([]byte, 15)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetCorrelationID(i string) (string, error) {
	now := GetEpochNow()
	seed := i + strconv.FormatInt(now, 10)
	hash := sha1.Sum([]byte(seed))
	result := fmt.Sprintf("%x", hash)
	return result, nil
}

func GetFormattedNow() string {
	t := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	return formatted
}

func AddTimeToNowEpoch(years int, months int, days int) int64 {
	t := time.Now()
	t2 := t.AddDate(years, months, days)
	return t2.Unix()
}

func GetEpochNow() int64 {
	return time.Now().Unix()
}

func TurnUnixTimestampToString(u int64) string {
	uS := strconv.FormatInt(u, 10)
	i, err := strconv.ParseInt(uS, 10, 64)
	if err != nil {
		panic(err)
	}
	tm := time.Unix(i, 0)
	return tm.String()
}

func GetRandomSerial() *big.Int {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))
	n, _ := rand.Int(rand.Reader, max)
	return n
}
