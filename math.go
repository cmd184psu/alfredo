package alfredo

import (
	"math/rand"
	"time"
)

//need to calculate the rate of change for units over time
// this is used to calculate the ETA and throughput
// the rate of change is calculated as units processed over time

func CalculateRateOfChange(unitsProcessed int64, startTime int64, endTime int64) float64 {
	VerbosePrintf("BEGIN CalculateRateOfChange(%d, %d)\n", unitsProcessed, startTime)
	defer VerbosePrintf("END CalculateRateOfChange(%d, %d)\n", unitsProcessed, startTime)

	if endTime == 0 {
		endTime = int64(time.Now().Unix())
	}
	if unitsProcessed == 0 || startTime == 0 || endTime <= startTime {
		return 0
	}
	// fmt.Printf("units processed: %d\n", unitsProcessed)
	// fmt.Printf("seconds elapsed: %d\n", now-startTime)
	// fmt.Printf("roc: %f\n", float64(unitsProcessed)/(float64(now)-float64(startTime)))

	roc := float64(unitsProcessed) / (float64(endTime) - float64(startTime))
	if roc < 0.00001 {
		roc = 0
	}
	return roc
}

func CalculateETARaw(remaining int64, rateOfChange float64) int64 {
	if rateOfChange == 0 {
		return 0
	}
	if remaining == 0 {
		return 0
	}
	eta := int64(float64(remaining) / rateOfChange)
	if eta < 1 {
		eta = 1
	}
	return eta
}

func GetRandomInt64InRange(min, max int64) int64 {
	if min >= max {
		return min
	}
	return min + rand.Int63n(max-min+1)
}
