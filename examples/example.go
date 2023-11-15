package main

import (
	"context"
	"github.com/sirupsen/logrus"
	rlog "rotateloghook/pkg/rotateloghook"
	"time"
)

func main() {

	log := logrus.New()
	//Time patterns specs: https://pkg.go.dev/github.com/lestrrat-go/strftime#readme-supported-conversion-specifications
	// Warning! time patterns of log file uses for rotation logic!
	hook, err := rlog.NewHook("./log_%Y-%m-%d-%H:%M:%S.log",
		//rlog.WithMaxAge(time.Second*25), // Old logs will be purged
		//rlog.WithRotationCount(7), // Count of storing logfiles
		rlog.WithRotationTime(time.Second*10), // Rotate after period. Warning:Period precision MUST be presented in filename time pattern!
		//rlog.WithClock(rlog.UTC), // timezone, may be omitted
		rlog.WithRotationSize(5*1024*1024), // Max file size in bytes
		rlog.WithLinkName("./CommonLog"),   // Can create fresher symbolyc link on actual log file. Correct working only in root folder
	)
	if err != nil {
		log.Panicf("Can't initialize rotation logger: %s", err)
	}

	hook.SetFormatter(&logrus.JSONFormatter{})

	// Hook after every rotation
	hook.SubscribeToRotation(func(evt rlog.RotationEvent) {
		log.Warningf("RotationHandler! New file is: %s, Old: ", evt.NewFileName, evt.PrevFileName)
	})

	log.Hooks.Add(hook)

	ctx := context.Background()

	go func() {
		iter := 0
		for {
			iter++
			log.Info("Infinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop messageInfinite loop message")
			if iter < 100 {
				time.Sleep(time.Second)
			} else {
				time.Sleep(time.Millisecond * 500)
			}
		}
	}()

	select {
	case <-ctx.Done():
		{
		}
	}

	log.Warningf("Test %v", 777)
}
